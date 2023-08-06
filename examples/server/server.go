package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/serboupal/iris"
)

var hub *iris.Hub

const defHost = "localhost:8888"

func main() {

	http.HandleFunc("/stats", stats)
	http.HandleFunc("/helo", helo)

	srv := &http.Server{
		Addr:              defHost,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           http.DefaultServeMux,
	}

	hub = iris.NewHub(defHost)
	hub.OringinFunc = func(r *http.Request) bool {
		return true
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			hub.Exit <- err
		}
	}()

	err := hub.Listen(context.Background())
	if err != nil {
		panic(err)
	}
}

func helo(w http.ResponseWriter, r *http.Request) {
	c, err := hub.ConnectPeer(w, r)
	if err != nil {
		return
	}
	m := iris.NewCmd(nil, iris.Hello, []string{c.Id.String()})
	c.Send(m)
}

func stats(w http.ResponseWriter, r *http.Request) {
	s := hub.Statics()

	data, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
