package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	hub = iris.NewHub(defHost, 5*time.Second, 5*time.Second)
	hub.OringinFunc = func(r *http.Request) bool {
		return true
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	hub.DoFunc = func(p *iris.Peer, c iris.Msg) {
		fmt.Println("rcv ", c, "from peer", p.Id)
		switch c.Data[0] {
		case "hello":
			fmt.Println("sending goodbye")
			p.Send(iris.Msg{
				Data: []string{"goodbye"},
			})
		}
	}

	err := hub.Listen(context.Background())
	if err != nil {
		panic(err)
	}
}

func helo(w http.ResponseWriter, r *http.Request) {
	_, err := hub.ConnectPeer(w, r)
	if err != nil {
		return
	}
}

func stats(w http.ResponseWriter, r *http.Request) {
	s := hub.Statics("iris")
	w.Header().Add("Content-Type", "application/json")
	data, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
