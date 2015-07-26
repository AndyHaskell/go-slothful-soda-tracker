package main

import (
	"net/http"
)

func main() {
	broadcaster := initMapBroadcaster()
	mux := initMux(broadcaster)

	svr := &http.Server{
		Addr:    ":1123",
		Handler: mux,
	}

	svr.ListenAndServe()
}
