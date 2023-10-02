package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/serboupal/iris"
)

const defHost = "localhost:8888"

func main() {
	const endpoint = "ws://" + defHost + "/helo"
	c := iris.NewClient(endpoint, 6*time.Second, 6*time.Second)
	c.Subscribe("test-chan")
	c.Publish("test-chan", []string{"test message"})
	c.Produce("proc-data-get", []string{"2323", "2023", "XXX"})

	c.DoFunc = func(data iris.Msg) {
		fmt.Println("received data", data)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		panic(c.Listen(context.Background()))
	}()

	c.Send(iris.Msg{
		Data: []string{"hello"},
	})

	wg.Wait()
}
