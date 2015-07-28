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

//broadcast takes in a message and sends it to the receiveMsgs channel of every
//user in the MapBroadcaster's Users map.
func (m *MapBroadcaster) broadcast(msg []byte) {
	for _, user := range m.Users {
		user.msgFromBroadcaster(msg)
	}
}

//broadcastToEveryoneBut takes in a User's ID string and a message and sends the
//message to every user in the MapBroadcaster's Users map except the user with
//the ID passed in.
func (m *MapBroadcaster) broadcastToEveryoneBut(id string, msg []byte) {
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
					m.broadcastToEveryoneBut(coords.Id, sendCoords)
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
				m.broadcastToEveryoneBut(idString, joinMsg)
			case id := <-m.disconnect:
				if _, ok := m.Users[id]; ok {
					m.Users[id].msgFromBroadcaster([]byte("close"))
					disconnectMsg, _ := MakeMsg("User disconnected", id)
					m.broadcastToEveryoneBut(id, disconnectMsg)
					delete(m.Users, id)
				}
			}
		}
	}()
}
