// server.go
package main

import (
	ws "github.com/gorilla/websocket"
	"net/http"
)

//MakeStaticFileRoute takes in a net/http ServeMux and strings for URL and
//directory paths and registers a route on the ServeMux for the URL path
//statically serving files from the directory path.
func MakeStaticFileRoute(m *http.ServeMux, urlPath, dirPath string) {
	m.Handle(urlPath, http.StripPrefix(urlPath,
		http.FileServer(http.Dir(dirPath))))
}

//serveMainPage serves the index.html page in the views directory.
func serveMainPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "views/index.html")
}

//initServeSocket takes in a Broadcaster and initializes a serveSocket net/http
//handler function that upgrades the request to create a new WebSocket Conn,
//which then is used added to the Broadcaster.
func initServeSocket(b Broadcaster) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		up := ws.Upgrader{}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		b.AddConn(conn)
	}
}

//initMux takes in a Broadcaster and creates and returns a net/http ServeMux for
//serving the web app, with the main page on the / route and the WebSocket
//connections initialized on the /ws route. initMux also starts the
//Broadcaster's ManageUsers goroutine.
func initMux(b Broadcaster) *http.ServeMux {
	mux := http.NewServeMux()

	MakeStaticFileRoute(mux, "/images/", "public/images")
	MakeStaticFileRoute(mux, "/styles/", "public/styles")
	MakeStaticFileRoute(mux, "/scripts/", "public/scripts")

	mux.HandleFunc("/", serveMainPage)
	mux.HandleFunc("/ws", initServeSocket(b))
	b.ManageUsers()

	return mux
}
