package main

import (
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
)

var dialer = ws.Dialer{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type EmptyBroadcaster struct{}

func (*EmptyBroadcaster) AddConn(c *ws.Conn)       {}
func (*EmptyBroadcaster) ManageUsers()             {}
func (*EmptyBroadcaster) DisconnectConn(id string) {}
func (*EmptyBroadcaster) MsgFromConn(msg []byte)   {}

//Variations on the test helpers used for testing Negroni
//(github.com/codegangsta/negroni) that call Fatalf rather than Errorf when the
//tests fail.
func expectF(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refuteF(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Fatalf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

//Construct the /ws WebSocket URL that connects to the WebSocket server.
func getWebSocketURL(svr *httptest.Server) string {
	urlNoProtocol := strings.TrimPrefix(svr.URL, "http")
	return "ws" + urlNoProtocol + "/ws"
}

//initServerWithEmptyBroadcaster initializes an httptest server with the routes
//defined in initMux and an EmptyBroadcaster as the server's Broadcaster and
//returns the server.
func initServerWithEmptyBroadcaster() *httptest.Server {
	return httptest.NewServer(initMux(&EmptyBroadcaster{}))
}

//initTestServer initializes a MapBroadcaster and an httptest server that uses
//that MapBroadcaster and the routes defined in initMux and returns the server
//and the MapBroadcaster.
func initTestServer() (*httptest.Server, *MapBroadcaster) {
	m := initMapBroadcaster()
	svr := httptest.NewServer(initMux(m))
	return svr, m
}

//makeWebSocketConn takes in a testing.T and an httptest Server and returns a
//WebSocket connection to the server. If the connection fails, the test stops,
//sending an error for the response's HTTP status (supposed to be 101) and
//the text of the connection error.
func makeWebSocketConn(t *testing.T, svr *httptest.Server) *ws.Conn {
	wsURL := getWebSocketURL(svr)
	conn, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}

	return conn
}

//readMessagesAndKeepTheLastOne takes in a Gorilla websocket connection and how
//many messages to read and reads in that number of messages, returning the last
//message read if no error was encountered or an error if one was encountered.
func readMessagesAndKeepTheLastOne(conn *ws.Conn, howMany int) ([]byte, error) {
	var msg []byte
	var err error
	for i := 0; i < howMany; i++ {
		_, msg, err = conn.ReadMessage()
		if err != nil {
			return nil, err
		}
	}
	return msg, nil
}

//Test that a conection is started when connecting to ws
func TestConn(t *testing.T) {
	svr := initServerWithEmptyBroadcaster()
	defer svr.Close()

	conn := makeWebSocketConn(t, svr)
	conn.Close()
}

//Verify that two connections can be added to the broadcaster
func TestTwoConns(t *testing.T) {
	svr, broadcaster := initTestServer()
	defer svr.Close()

	//Make the first connection
	conn1 := makeWebSocketConn(t, svr)
	defer conn1.Close()

	//Make sure first connection was added to broadcaster
	expectF(t, broadcaster.highestId, 1)

	//Make the second connection
	conn2 := makeWebSocketConn(t, svr)
	defer conn2.Close()

	//Make sure second connection was added to broadcaster
	expectF(t, broadcaster.highestId, 2)
}

//Makes sure when a user joins the server they get a message with their ID
//number
func TestGetIdMessage(t *testing.T) {
	svr, _ := initTestServer()
	defer svr.Close()

	yourIdJSON := `{"msgType":"Your ID","data":"1"}`

	//Make a connection
	conn := makeWebSocketConn(t, svr)
	defer conn.Close()

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	expectF(t, string(msg), yourIdJSON)
}

//Makes sure when a user joins they get a message with the ID numbers and
//coordinates of all users
func TestGetEveryoneMessage(t *testing.T) {
	twoUsersJSON := `{"msgType":"Everyone","data":{"users":{` +
		`"1":{"id":"1","lat":0,"lng":0},` +
		`"2":{"id":"2","lat":0,"lng":0}}}}`

	svr, _ := initTestServer()
	defer svr.Close()

	//Make two connections
	conn1 := makeWebSocketConn(t, svr)
	defer conn1.Close()
	conn2 := makeWebSocketConn(t, svr)
	defer conn2.Close()

	everyone, err := readMessagesAndKeepTheLastOne(conn2, 2)
	expectF(t, err, nil)
	expectF(t, string(everyone), twoUsersJSON)
}

//Makes sure when a user disconnects they are removed from the broadcaster's
//Users map.
func TestDisconnect(t *testing.T) {
	svr, broadcaster := initTestServer()
	defer svr.Close()

	conn1 := makeWebSocketConn(t, svr)
	defer conn1.Close()
	conn2 := makeWebSocketConn(t, svr)
	conn2.Close()

	//Make sure second connection was removed from broadcaster
	time.Sleep(100)
	expectF(t, len(broadcaster.Users), 1)
}

//Makes sure when a user joins the server all users except the user that joined
//get a "user joined" message
func TestUserJoinedMessage(t *testing.T) {
	svr, _ := initTestServer()
	defer svr.Close()

	//Make the first connection
	conn1 := makeWebSocketConn(t, svr)
	defer conn1.Close()

	//Have the first connection read its ID and everyone messages
	_, err := readMessagesAndKeepTheLastOne(conn1, 2)
	expectF(t, err, nil)

	//Make the second connection
	conn2 := makeWebSocketConn(t, svr)
	defer conn2.Close()

	//Have the second connection read its ID and everyone messages
	_, err = readMessagesAndKeepTheLastOne(conn2, 2)
	expectF(t, err, nil)

	//Make sure the first user gets the message that the second user joined
	secondUserJoinedJSON := `{"msgType":"User joined","data":"2"}`
	_, msg, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	expectF(t, string(msg), secondUserJoinedJSON)

	//Make sure the message that the second user joined isn't broadcasted to the
	//first user.
	conn2.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, msg, err = conn2.ReadMessage()
	if err == nil {
		t.Fatalf("Expected error reading message, msg = %s", msg)
	}
}

//Makes sure when a user disconnects they are removed from the broadcaster's
//Users map.
func TestDisconnectBroadcast(t *testing.T) {
	svr, _ := initTestServer()
	defer svr.Close()

	//Make two connections
	conn1 := makeWebSocketConn(t, svr)
	defer conn1.Close()

	//Have the first connection read its ID and everyone messages
	_, err := readMessagesAndKeepTheLastOne(conn1, 2)
	expectF(t, err, nil)

	conn2 := makeWebSocketConn(t, svr)

	//Have the second connection read its ID and everyone messages
	_, err = readMessagesAndKeepTheLastOne(conn2, 2)
	expectF(t, err, nil)

	//Read in the message on the first connection that the second user connected
	_, _, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}

	conn2.Close()

	//Read in the message on the first connection that the second user
	//disconnected
	secondUserDisconnectedJSON := `{"msgType":"User disconnected","data":"2"}`
	_, msg, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	expectF(t, string(msg), secondUserDisconnectedJSON)
}

//Makes sure when a user sends a JSON message containing their ID number and
//coordinates that it is broadcasted to every user connected to the broadcaster
//except the user that sent the message.
func TestBroadcastCoords(t *testing.T) {
	svr, _ := initTestServer()
	defer svr.Close()

	//Make two connections
	conn1 := makeWebSocketConn(t, svr)
	defer conn1.Close()

	//Have the first connection read its ID and everyone messages
	_, err := readMessagesAndKeepTheLastOne(conn1, 2)
	expectF(t, err, nil)

	conn2 := makeWebSocketConn(t, svr)
	defer conn2.Close()

	//Have the second connection read its ID and everyone messages
	_, err = readMessagesAndKeepTheLastOne(conn2, 2)
	expectF(t, err, nil)

	//Have the first connection read the message that the second user joined
	_, _, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}

	firstUserAtFreshPondJSON :=
		`{"msgType":"Update coordinates","data":` +
			`{"id":"1","lat":42.388282,"lng":-71.153968}}`

	//Broadcast that the first user is at the Cambridge Fresh Pond
	conn1.WriteMessage(ws.TextMessage,
		[]byte(`{"id":"1","lat":42.388282,"lng":-71.153968}`))

	//Make sure the first user's location is sent to the second user and not the
	//first user.
	_, msg, err := conn2.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	expectF(t, string(msg), string(firstUserAtFreshPondJSON))

	//Make sure the first user's location wasn't broadcasted to the first user
	conn1.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, msg, err = conn1.ReadMessage()
	if err == nil {
		t.Fatalf("Expected error reading message, msg = %s", msg)
	}
}
