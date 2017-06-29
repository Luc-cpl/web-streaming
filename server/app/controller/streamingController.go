package controller

import (
	"fmt"
	"log"
	"net/http"

	"gopkg.in/mgo.v2/bson"

	"time"

	"encoding/json"

	"strings"

	"github.com/Luc-cpl/jsonMap"
	mgoS "github.com/Luc-cpl/mgoSimpleCRUD"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Hub struct {
	//striaming from client to user
	Client map[string]*Client

	//request from user to client
	User map[string]map[*Client]bool
}

var Streaming Hub

//NewHub creates a hub for comunication
func NewHub() Hub {
	return Hub{
		Client: make(map[string]*Client),
		User:   make(map[string]map[*Client]bool),
	}
}

//StartClientWebsocket start a websocket connection whith client using the mgoSimpleCRUD package
func StartClientWebsocket(w http.ResponseWriter, r *http.Request) {
	stopIDCheck := make(chan bool)
	defer func() {
		stopIDCheck <- true
	}()
	values := strings.Split(string(mux.Vars(r)["rest"]), "&")

	if len(values) != 2 {
		w.Write([]byte(`{"err":"the passed value does not match with necessary fields"}`))
		return
	}

	id, err := getID(values[0])
	if err != nil {
		w.Write([]byte(`{"err":"` + err.Error() + `"}`))
		return
	}
	name := values[1]

	if _, ok := Streaming.Client[id+" - "+name]; ok {
		w.Write([]byte(`{"err":"the connection already exist"}`))
		return
	}

	connName := id + " - " + name

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	client.User.ID = id
	go client.checkForNewIdentity(stopIDCheck)

	Streaming.Client[connName] = client

	client.readClientPump(connName)
}

func (c *Client) readClientPump(connName string) {
	stop := make(chan bool)              //a chan to stop the writePump
	go c.writeClientPump(connName, stop) //starts the writePump

	defer func() {
		delete(Streaming.Client, connName)
		delete(Streaming.User, connName)
		stop <- true
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		msg := []byte("teste")
		_, msg, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			var newMsg interface{}
			json.Unmarshal([]byte(`{"err":"`+err.Error()+`"}`), &newMsg)
			msg, _ := json.Marshal(newMsg)
			for user := range Streaming.User[connName] {
				select {
				case user.send <- msg:
				default:
					close(user.send)
					delete(Streaming.User[connName], user)
				}
			}
			break
		}

		//aqui envia a msg para cada usuário buscando esse websocket usuário
		for user := range Streaming.User[connName] {
			select {
			case user.send <- msg:
			default:
				close(user.send)
				delete(Streaming.User[connName], user)
			}
		}

	}
}

func (c *Client) writeClientPump(connName string, stop chan bool) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case <-stop:
			return
		case resp, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			var msg interface{}
			json.Unmarshal(resp, &msg)
			c.conn.WriteJSON(msg)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

//StartUserWebsocket starts the websocket for the client
func StartUserWebsocket(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["rest"]
	user := GetUser(r)
	connName := user.ID + " - " + name

	if _, ok := Streaming.Client[connName]; !ok {
		w.Write([]byte(`{"err":"this connection doesn't exist"}`))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	client.User = user

	if _, ok := Streaming.User[connName]; !ok {
		Streaming.User[connName] = make(map[*Client]bool)
	}

	Streaming.User[connName][client] = true

	client.readUserPump(connName)

}

func (c *Client) readUserPump(connName string) {
	stop := make(chan bool)                    //a chan to stop the writePump
	errMsg := make(chan interface{})           //a chan to stop the writePump
	go c.writeUserPump(connName, errMsg, stop) //starts the writePump

	defer func() {
		delete(Streaming.User[connName], c)
		if len(Streaming.User[connName]) == 0 {
			delete(Streaming.User, connName)
		}
		stop <- true
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, msg, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			var newMsg interface{}
			json.Unmarshal([]byte(`{"err":"`+err.Error()+`"}`), &newMsg)
			msg, _ = json.Marshal(newMsg)
			errMsg <- msg
			break
		}
		Streaming.Client[connName].send <- msg

	}
}

func (c *Client) writeUserPump(connName string, errMsg chan interface{}, stop chan bool) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case <-stop:
			return
		case resp, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(1, resp)

		case resp, ok := <-errMsg:
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

//GetClients shows all clients open for a user acconunt
func GetClients(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	jsonStr := `[`
	for key := range Streaming.Client {
		if strings.HasPrefix(key, user.ID) {
			jsonStr += `"` + strings.Trim(key, user.ID+` - `) + `"`
			if !strings.EqualFold(jsonStr, `[`) {
				jsonStr += `,`
			}
		}
	}
	jsonStr += `]`

	w.Write([]byte(jsonStr))
}

func getID(userIdentity string) (id string, err error) {
	newSession := mgoS.DB.Session.Copy()
	defer newSession.Close()

	checkByt := []byte(`{"` + mgoS.DB.UserIdentityValue + `": "` + userIdentity + `"}`)
	var checkInterface interface{}
	err = json.Unmarshal(checkByt, &checkInterface)

	var jsonInterface interface{}
	err = newSession.DB(mgoS.DB.Database).C("users").Find(checkInterface).One(&jsonInterface)

	if err != nil {
		return "", err
	}

	byt, _ := json.Marshal(jsonInterface)
	u, _ := jsonMap.GetMap(byt, "")
	id = strings.Trim(u["_id"], `"`)

	return id, nil
}

func (c *Client) checkForNewIdentity(stop chan bool) {
	send := false
	var oldIdentity string

	for {
		select {
		case <-stop:
			return
		default:
			var jsonInterface interface{}
			newSession := mgoS.DB.Session.Copy()
			err := newSession.DB(mgoS.DB.Database).C("users").FindId(bson.ObjectIdHex(c.User.ID)).One(&jsonInterface)
			newSession.Close()
			if err == nil {
				byt, _ := json.Marshal(jsonInterface)
				u, _ := jsonMap.GetMap(byt, "")
				newIdentity := strings.Trim(u[mgoS.DB.UserIdentityValue], `"`)

				if oldIdentity != newIdentity {
					if send {
						oldIdentity = newIdentity
						c.send <- []byte(`{"` + mgoS.DB.UserIdentityValue + `": "` + oldIdentity + `"}`)
					}
					send = true
				}
			} else {
				return
			}
			time.Sleep(time.Second * 300)
		}

	}

}
