package controller

import (
	"log"
	"net/http"

	"time"

	"encoding/json"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Hub struct {
	//striaming from client to user
	Client map[string]chan []byte
	//the key contains the name of the active clients
	ClientLog map[string]bool
	//request from user to client
	User map[string]map[*Client]bool
}

var Streaming Hub

//NewHub creates a hub for comunication
func NewHub() Hub {
	return Hub{
		Client:    make(map[string]chan []byte),
		ClientLog: make(map[string]bool),
		User:      make(map[string]map[*Client]bool),
	}
}

//StartClientWebsocket start a websocket connection whith client using the mgoSimpleCRUD package
func StartClientWebsocket(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["rest"]
	user := GetUser(r)
	if _, ok := Streaming.ClientLog[user.ID+" - "+name]; ok {
		w.Write([]byte(`{"err":"the connection already exist"}`))
		return
	}

	Streaming.ClientLog[user.ID+" - "+name] = true

	connName := user.ID + " - " + name

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	client.User = user

	client.readClientPump(connName)
}

func (c *Client) readClientPump(connName string) {
	stop := make(chan bool)              //a chan to stop the writePump
	go c.writeClientPump(connName, stop) //starts the writePump

	defer func() {
		delete(Streaming.ClientLog, connName)
		delete(Streaming.Client, connName)
		delete(Streaming.User, connName)
		stop <- true
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
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
		case resp, ok := <-Streaming.Client[connName]:
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

//StartUserWebsocket starts the websocket for the client
func StartUserWebsocket(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["rest"]
	user := GetUser(r)
	connName := user.ID + " - " + name

	if _, ok := Streaming.ClientLog[connName]; !ok {
		w.Write([]byte(`{"err":"this connection doesn't exist"}`))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	client.User = user

	Streaming.User[connName][client] = true

	client.readUserPump(connName)

}

func (c *Client) readUserPump(connName string) {
	stop := make(chan bool)                    //a chan to stop the writePump
	errMsg := make(chan interface{})           //a chan to stop the writePump
	go c.writeUserPump(connName, errMsg, stop) //starts the writePump

	defer func() {
		delete(Streaming.User[connName], c)
		stop <- true
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
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
			msg, _ := json.Marshal(newMsg)
			errMsg <- msg
			break
		}

		Streaming.Client[connName] <- msg

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
			c.conn.WriteJSON(resp)

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
