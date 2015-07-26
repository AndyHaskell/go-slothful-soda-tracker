package main

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
)

var dialer = ws.Dialer{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type EmptyBroadcaster struct {}
func (*EmptyBroadcaster) AddConn(c *ws.Conn){}
func (*EmptyBroadcaster) ManageUsers(){}
func (*EmptyBroadcaster) DisconnectConn(id string){}
func (*EmptyBroadcaster) MsgFromConn(msg []byte){}

//Construct the /ws WebSocket URL that connects to the WebSocket server.
func getWebSocketURL(svr *httptest.Server) string {
	urlNoProtocol := strings.TrimPrefix(svr.URL, "http")
	return "ws" + urlNoProtocol + "/ws"
}

func initServerWithEmptyBroadcaster() *httptest.Server{
	return httptest.NewServer(initMux(&EmptyBroadcaster{}))
}

func initTestServer() (*httptest.Server, *MapBroadcaster) {
	m := initMapBroadcaster()
	svr := httptest.NewServer(initMux(m))
	return svr, m
}

//Test that a conection is started when connecting to ws
func TestConn(t *testing.T) {
	svr := initServerWithEmptyBroadcaster()
	defer svr.Close()

	wsURL := getWebSocketURL(svr)
	conn, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn.Close()
}

//Verify that two connections can be added to the broadcaster
func TestTwoConns(t *testing.T) {
	svr, broadcaster := initTestServer()
	defer svr.Close()

	wsURL := getWebSocketURL(svr)

	//Make the first connection
	conn1, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn1.Close()

	//Make sure first connection was added to broadcaster
	if broadcaster.highestId != 1 {
		t.Fatalf("broadcaster.highestId: Expected %d, got %d",
			1, broadcaster.highestId)
	}

	//Make the second connection
	conn2, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn2.Close()

	//Make sure second connection was added to broadcaster
	if broadcaster.highestId != 2 {
		t.Fatalf("broadcaster.highestId: Expected %d, got %d",
			2, broadcaster.highestId)
	}
}

//Makes sure when a user joins the server they get a message with their ID
//number
func TestGetIdMessage(t *testing.T) {
	svr, _ := initTestServer()
	defer svr.Close()

	wsURL := getWebSocketURL(svr)

	//Make a connection
	conn, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn.Close()

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	if string(msg) != `{"msgType":"Your ID","data":"1"}` {
		t.Fatalf("msg: Expected %s, got %s", `{"msgType":"Your ID","data":"1"}`, msg)
	}
}

//Makes sure when a user joins they get a message with the ID numbers and
//coordinates of all users
func TestGetEveryoneMessage(t *testing.T) {
	twoUsersJSON := `{"msgType":"Everyone","data":{"users":{` +
		`"1":{"id":"1","lat":0,"lng":0},` +
		`"2":{"id":"2","lat":0,"lng":0}}}}`

	svr, _ := initTestServer()
	defer svr.Close()

	wsURL := getWebSocketURL(svr)

	//Make two connections
	conn1, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn1.Close()
	conn2, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn2.Close()

	_, _, err = conn2.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	_, everyone, err := conn2.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	if string(everyone) != twoUsersJSON {
		t.Fatalf("everyone: Expected %s, got %s", twoUsersJSON,  everyone)
	}
}


//Makes sure when a user disconnects they are removed from the broadcaster's
//Users map.
func TestDisconnect(t *testing.T) {
	svr, broadcaster := initTestServer()
	defer svr.Close()

	wsURL := getWebSocketURL(svr)

	//Make two connections
	conn1, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn1.Close()
	conn2, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	conn2.Close()

	//Make sure second connection was added to broadcaster
	time.Sleep(100)
	if len(broadcaster.Users) != 1 {
		t.Errorf("len(broadcaster.Users): Expected %d, got %d",
			1, len(broadcaster.Users))
	}
}

//Makes sure when a user joins the server all users except the user that joined
//get a "user joined" message
func TestUserJoinedMessage(t *testing.T) {
	svr, _ := initTestServer()
	defer svr.Close()

	wsURL := getWebSocketURL(svr)

	//Make the first connection
	conn1, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn1.Close()

	//Have the first connection read its ID and everyone messages
	_, msg, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	_, msg, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}

	//Make the second connection
	conn2, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn2.Close()

	//Have the second connection read its ID and everyone messages
	_, msg, err = conn2.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	_, msg, err = conn2.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}

	//Make sure the first user gets the message that the second user joined
	secondUserJoinedJSON := `{"msgType":"User joined","data":"2"}`
	_, msg, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	if string(msg) != secondUserJoinedJSON {
		t.Fatalf("msg: Expected %s, got %s", secondUserJoinedJSON, msg)
	}

	//Make sure the message that the second user joined isn't broadcasted to the
	//first user.
	conn2.SetReadDeadline(time.Now().Add(500*time.Millisecond))
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

	wsURL := getWebSocketURL(svr)

	//Make two connections
	conn1, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn1.Close()

	//Have the first connection read its ID and everyone messages
	_, msg, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}

	_, msg, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}

	conn2, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}

	//Have the second connection read its ID and everyone messages
	_, msg, err = conn2.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	_, msg, err = conn2.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	
	//Read in the message on the first connection that the second user connected
	_, msg, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}

	conn2.Close()
	
	//Read in the message on the first connection that the second user
	//disconnected
	secondUserDisconnectedJSON := `{"msgType":"User disconnected","data":"2"}`
	_, msg, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	if string(msg) != secondUserDisconnectedJSON {
		t.Fatalf("msg: Expected %s, got %s", secondUserDisconnectedJSON, msg)
	}
}

func TestBroadcastCoords(t *testing.T){
	svr, _ := initTestServer()
	defer svr.Close()

	wsURL := getWebSocketURL(svr)

	//Make two connections
	conn1, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn1.Close()
	
	//Have the first connection read its ID and everyone messages
	_, msg, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}

	_, msg, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}

	conn2, res, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("%v", res.Status)
		t.Fatalf("Dial failed, %v", err)
	}
	defer conn2.Close()

	//Have the second connection read its ID and everyone messages
	_, msg, err = conn2.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	_, msg, err = conn2.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	_, msg, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}

	firstUserAtFreshPondJSON :=
		`{"msgType":"Update coordinates","data":`+
			`{"id":"1","lat":42.388282,"lng":-71.153968}}`

	conn1.WriteMessage(ws.TextMessage,
		[]byte(`{"id":"1","lat":42.388282,"lng":-71.153968}`))

	//Make sure the first user's location is sent to the second user and not the
	//first user.
	_, msg, err = conn2.ReadMessage()
	if err != nil {
		t.Fatalf("Read message failed, %v", err)
	}
	if string(msg) != string(firstUserAtFreshPondJSON) {
		t.Fatalf("msg: Expected %s, got %s", firstUserAtFreshPondJSON, msg)
	}

	//Make sure the first user's location wasn't broadcasted to the first user
	conn1.SetReadDeadline(time.Now().Add(500*time.Millisecond))
	_, msg, err = conn1.ReadMessage()
	if err == nil {
		t.Fatalf("Expected error reading message, msg = %s", msg)
	}
}