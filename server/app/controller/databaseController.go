package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"time"

	"reflect"

	mgoS "github.com/Luc-cpl/mgoSimpleCRUD"
	"github.com/gorilla/websocket"
)

type userCall struct {
	Request     []mgoS.Request `json:"request"`
	LoopRequest bool           `json:"loopRequest"` //false = request one time | true = request continuously (only for READ)
	BreakLoop   bool           `json:"breakLoop"`   //false = request one time | true = request continuously (only for READ)
}

const (
	writeWait      = 10 * time.Second    // Time allowed to write a message to the peer.
	pongWait       = 60 * time.Second    // Time allowed to read the next pong message from the peer.
	pingPeriod     = (pongWait * 9) / 10 // Send pings to peer with this period. Must be less than pongWait.
	maxMessageSize = 512                 // Maximum message size allowed from peer.
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	conn *websocket.Conn // The websocket connection.
	send chan []byte     // Buffered channel of outbound messages.
	User mgoS.User       // The user information.
}

//NewWebsocket start a websocket connection whith client using the mgoSimpleCRUD package
func NewWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	client.User = GetUser(r)

	client.readPump()

}

func (c *Client) readPump() {
	messages := make(chan []interface{}) //a chan to pass messages to writePump
	stop := make(chan bool)              //a chan to stop the writePump
	stopLoop := make(chan bool)          //a chan to stop all the loopRequests

	go c.writePump(messages, stop) //starts the writePump

	defer func() {
		stop <- true
		stopLoop <- true
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	nLoopsChain := make(chan int)
	nLoops := 0
	for {
		_, msg, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		var call userCall
		err = json.Unmarshal(msg, &call)
		if err != nil {
			var newMsg []interface{}
			json.Unmarshal([]byte(`[{"err":"`+err.Error()+`"}]`), &newMsg)
			messages <- newMsg
		}

		if call.BreakLoop {
			for nLoops > 0 {
				stopLoop <- true
				nLoops += <-nLoopsChain
			}
		}

		if !call.LoopRequest {
			go c.singleRequest(call, messages)
		} else {
			n := 0
			for key := range call.Request {
				if strings.Contains("find findOne readID", call.Request[key].Method) {
					n++
				}
			}
			if n == len(call.Request) {
				go c.startLoop(call, messages, nLoopsChain, stopLoop)
				nLoops += <-nLoopsChain
			} else {
				var newMsg []interface{}
				json.Unmarshal([]byte(`[{"err":"can't loop into non read request"}]`), &newMsg)
				messages <- newMsg
			}
		}

	}
}

func (c *Client) writePump(messages chan []interface{}, stop chan bool) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case <-stop:
			return
		case resp, ok := <-messages:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.WriteJSON(resp)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Client) startLoop(call userCall, messages chan []interface{}, nLoops chan int, stopLoop chan bool) {
	var oldResp []interface{}
	defer func() {
		nLoops <- -1
	}()
	first := true
	for {
		select {
		case <-stopLoop:
			return
		default:
			resp, err := check(c.User, call.Request)
			if err != nil {
				var newMsg []interface{}
				json.Unmarshal([]byte(`[{"err":"`+err.Error()+`"}]`), &newMsg)
				messages <- newMsg
				if first {
					return
				}
			} else if !reflect.DeepEqual(oldResp, resp) {
				if first {
					first = false
					nLoops <- 1
				}
				oldResp = resp
				messages <- resp
			}
		}
	}
}

func (c *Client) singleRequest(call userCall, messages chan []interface{}) {
	resp, err := check(c.User, call.Request)
	if err != nil {
		var newMsg []interface{}
		json.Unmarshal([]byte(`[{"err":"`+err.Error()+`"}]`), &newMsg)
		messages <- newMsg
		return
	}
	messages <- resp
	return
}

func check(user mgoS.User, request []mgoS.Request) (response []interface{}, err error) {
	respInter := make([]interface{}, len(request))
	for key := range request {
		resp, err := mgoS.DB.CRUDRequest(user, request[key])
		if err != nil {
			return nil, err
		}
		json.Unmarshal(resp, &respInter[key])
	}
	return respInter, nil
}

func getRequest(request string, user mgoS.User) (response string) {
	var req []mgoS.Request
	err := json.Unmarshal([]byte(request), &req)
	if err != nil {
		errMsg := `[{"err":"` + err.Error() + `"}]`
		return errMsg
	}
	for key := range req {
		if !strings.Contains("find findOne readID", req[key].Method) {
			errMsg := `[{"err":"unnautorized method"}]`
			return errMsg
		}
	}

	responseInter, err := check(user, req)
	if err != nil {
		errMsg := `[{"err":"` + err.Error() + `"}]`
		return errMsg
	}

	responseByt, _ := json.Marshal(responseInter)
	return string(responseByt)
}
