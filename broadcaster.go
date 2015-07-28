package main

import (
	"encoding/json"
	"log"
	"strconv"

	ws "github.com/gorilla/websocket"
)

type Msg struct {
	MsgType string      `json:"msgType"`
	Data    interface{} `json:"data"`
}

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type LatLngMsg struct {
	Id  string  `json:"id"`
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

//MakeMsg takes in a string saying what type of message is being sent and some
//data in some type that can be converted to JSON and returns a JSON encoding
//of the data with a "msgType" attribute added. If the data can't be converted
//to JSON, an error is returned.
func MakeMsg(msgType string, data interface{}) ([]byte, error) {
	msg := Msg{
		MsgType: msgType,
		Data:    data,
	}

	return json.Marshal(msg)
}

type User struct {
	IdNumber    string `json:"id"`
	conn        *ws.Conn
	receiveMsgs chan []byte
	closed      bool
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

//initUser takes in an ID string, a Gorilla WebSocket Conn, and a MapBroadcaster
//pointer and initializes a User struct and has it receive from its client
//connection and the MapBroadcaster.
func initUser(id string, conn *ws.Conn, m *MapBroadcaster) *User {
	user := &User{
		IdNumber:    id,
		conn:        conn,
		receiveMsgs: make(chan []byte),
		closed:      false,
		Lat:         0.0,
		Lng:         0.0,
	}
	user.receiveFromBroadcaster(m)
	user.receiveFromClient(m)

	return user
}

//msgFromBroadcaster adds a message from a Broadcaster to the User's receiveMsgs
//channel.
func (user *User) msgFromBroadcaster(msg []byte) {
	user.receiveMsgs <- msg
}

//receiveFromBroadcaster runs as a goroutine and gets messages from the User's
//receiveMsgs channel, writing them to the client connection. If there is an
//error writing the message, the User closes and the goroutine ends.
func (user *User) receiveFromBroadcaster(b Broadcaster) {
	go func() {
		for {
			msg := <-user.receiveMsgs
			if string(msg) == "close" {
				user.conn.Close()
				close(user.receiveMsgs)
			}

			err := user.conn.WriteMessage(ws.TextMessage, msg)
			if err != nil {
				b.DisconnectConn(user.IdNumber)
				return
			}
		}
	}()
}

//receiveFromClient starts a goroutine for the User to listen for messages on
//from its client connection. If an error is encountered reading a message, such
//as an EOF from the client disconnecting, the User disconnects from the
//Broadcaster
func (user *User) receiveFromClient(b Broadcaster) {
	go func() {
		for {
			_, msg, err := user.conn.ReadMessage()
			if err != nil {
				b.DisconnectConn(user.IdNumber)
				return
			}

			b.MsgFromConn(msg)
		}
	}()
}

//updateCorods takes in a pair of latitude and longitude coordinates and sets
//the User's latitude and longitude coordinates to them
func (user *User) updateCoords(lat, lng float64) {
	user.Lat = lat
	user.Lng = lng
}

//getCoords creates a LatLngMsg struct containing a User's ID number and
//latitude and longitude coordinates.
func (user *User) getCoords() LatLngMsg {
	return LatLngMsg{
		Id:  user.IdNumber,
		Lat: user.Lat,
		Lng: user.Lng,
	}
}

type Broadcaster interface {
	AddConn(*ws.Conn)
	DisconnectConn(string)
	MsgFromConn([]byte)
	ManageUsers()
}

type MapBroadcaster struct {
	highestId  int
	Users      map[string]*User `json:"users"`
	join       chan *ws.Conn
	disconnect chan string
	messages   chan []byte
}

//initMapBroadcaster creates a new MapBroadcaster object.
func initMapBroadcaster() *MapBroadcaster {
	m := &MapBroadcaster{
		highestId:  0,
		Users:      make(map[string]*User),
		join:       make(chan *ws.Conn),
		disconnect: make(chan string),
		messages:   make(chan []byte),
	}
	return m
}

//AddConn takes in a Gorilla WebSocket connection and puts it on the
//MapBroadcaster's join channel.
func (m *MapBroadcaster) AddConn(c *ws.Conn) {
	m.join <- c
}

//DisconnectConn takes in the ID string of a User disconnecting from the
//MapBroadcaster and adds the ID string to the Broadcaster's disconnect channel.
func (m *MapBroadcaster) DisconnectConn(id string) {
	m.disconnect <- id
}

//
func (m *MapBroadcaster) MsgFromConn(msg []byte) {
	m.messages <- msg
}

//Broadcast takes in a message and sends it to the receiveMsgs channel of every
//user in the MapBroadcaster's Users map.
func (m *MapBroadcaster) Broadcast(msg []byte) {
	for _, user := range m.Users {
		user.msgFromBroadcaster(msg)
	}
}

//BroadcastToEveryoneBut takes in a User's ID string and a message and sends the
//message to every user in the MapBroadcaster's Users map except the user with
//the ID passed in.
func (m *MapBroadcaster) BroadcastToEveryoneBut(id string, msg []byte) {
	for currentId, user := range m.Users {
		if currentId != id {
			user.msgFromBroadcaster(msg)
		}
	}
}

//ManageUsers runs a goroutine that has a MapBroadcaster handle users
//connecting, disconnecting, and sending messages about their coordinates over
//the MapBroadcaster.
func (m *MapBroadcaster) ManageUsers() {
	go func() {
		for {
			select {
			case msg := <-m.messages:
				var coords LatLngMsg
				err := json.Unmarshal(msg, &coords)
				if err != nil {
					log.Println(err)
					continue
				}

				if _, ok := m.Users[coords.Id]; ok {
					m.Users[coords.Id].updateCoords(coords.Lat, coords.Lng)
					sendCoords, _ :=
						MakeMsg("Update coordinates",
							m.Users[coords.Id].getCoords())
					m.BroadcastToEveryoneBut(coords.Id, sendCoords)
				}
			case conn := <-m.join:
				m.highestId++
				idString := strconv.Itoa(m.highestId)
				user := initUser(idString, conn, m)
				m.Users[idString] = user

				yourIdMsg, _ := MakeMsg("Your ID", idString)
				everyoneMsg, _ := MakeMsg("Everyone", m)
				joinMsg, _ := MakeMsg("User joined", idString)

				user.msgFromBroadcaster(yourIdMsg)
				user.msgFromBroadcaster(everyoneMsg)
				m.BroadcastToEveryoneBut(idString, joinMsg)
			case id := <-m.disconnect:
				if _, ok := m.Users[id]; ok {
					m.Users[id].msgFromBroadcaster([]byte("close"))
					disconnectMsg, _ := MakeMsg("User disconnected", id)
					m.BroadcastToEveryoneBut(id, disconnectMsg)
					delete(m.Users, id)
				}
			}
		}
	}()
}
