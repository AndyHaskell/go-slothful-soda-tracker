package main

import (
	"encoding/json"

	ws "github.com/gorilla/websocket"
)

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

//MsgFromBroadcaster adds a message from a Broadcaster to the User's receiveMsgs
//channel.
func (user *User) MsgFromBroadcaster(msg []byte) {
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
		var coords LatLng

		for {
			_, msg, err := user.conn.ReadMessage()
			if err != nil {
				b.DisconnectConn(user.IdNumber)
				return
			}

			err = json.Unmarshal(msg, &coords)
			if err != nil {
				b.DisconnectConn(user.IdNumber)
				return
			}

			//Add the user's ID number to the JSON message before sending it to
			//the Broadcaster
			sendCoords := &LatLngMsg{
				Id:  user.IdNumber,
				Lat: coords.Lat,
				Lng: coords.Lng,
			}
			msgToSend, err := json.Marshal(sendCoords)
			if err != nil {
				b.DisconnectConn(user.IdNumber)
				return
			}

			b.MsgFromConn(msgToSend)
		}
	}()
}

//UpdateCoords takes in a pair of latitude and longitude coordinates and sets
//the User's latitude and longitude coordinates to them
func (user *User) UpdateCoords(lat, lng float64) {
	user.Lat = lat
	user.Lng = lng
}

//GetCoords creates a LatLngMsg struct containing a User's ID number and
//latitude and longitude coordinates.
func (user *User) GetCoords() LatLngMsg {
	return LatLngMsg{
		Id:  user.IdNumber,
		Lat: user.Lat,
		Lng: user.Lng,
	}
}
