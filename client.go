package iris

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Peer
	DoFunc func(data Msg)

	readWait  time.Duration
	writeWait time.Duration

	send chan Msg
	pong chan bool
}

func NewClient(addr string, readWite time.Duration, writeWait time.Duration) *Client {
	return &Client{
		Peer: Peer{
			Addr: addr,
			data: make(chan Msg),
		},
		readWait:  readWite,
		writeWait: writeWait,
		send:      make(chan Msg, 100),
		pong:      make(chan bool),
	}
}

func (c *Client) Listen(ctx context.Context) error {
	var err error
	c.conn, _, err = websocket.DefaultDialer.Dial(c.Addr, nil)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	go c.read(cancel)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d := <-c.data:
			if c.DoFunc == nil {
				return ErrInternal
			}
			c.DoFunc(d)
		case d := <-c.send:
			err := c.sendConn(d, c.writeWait)
			if err != nil {
				cancel()
				return ErrInternal
			}
		case <-c.pong:
			err := c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			if err != nil {
				return err
			}
			err = c.conn.WriteMessage(websocket.PongMessage, nil)
			if err != nil {
				cancel()
				return ErrInternal
			}
		}
	}
}

func (c *Client) Send(v Msg) {
	c.send <- v
}

func (c *Client) Reply(data []string, from Msg) {
	v := Msg{}
	v.Dest = from.Source
	v.Id = from.Id
	v.Cmd = Data
	v.Data = data
	c.send <- v
}

func (c *Client) Subscribe(name string) {
	m := Msg{
		Cmd:  Subscribe,
		Data: []string{name},
	}
	c.send <- m
}

func (c *Client) Produce(name string, data []string) {
	m := Msg{
		Cmd:  Produce,
		Data: append([]string{name}, data...),
	}
	c.send <- m
}

func (c *Client) Publish(name string, data []string) {
	m := Msg{
		Cmd:  Publish,
		Data: append([]string{name}, data...),
	}
	c.send <- m
}

func (c *Client) Consume(name string) error {
	m := Msg{
		Cmd:  Consume,
		Data: []string{name},
	}
	c.send <- m
	return nil
}

func (c *Client) Leave(name string) {
	m := Msg{
		Cmd:  Leave,
		Data: []string{name},
	}
	c.send <- m
}

func (c *Client) read(cancel context.CancelFunc) {
	c.conn.SetPingHandler(func(string) error {
		err := c.conn.SetReadDeadline(time.Now().Add(c.readWait))
		if err != nil {
			return err
		}
		c.pong <- true
		return nil
	})

	for {
		var v Msg
		c.conn.SetReadDeadline(time.Now().Add(c.readWait))
		err := c.conn.ReadJSON(&v)
		if err != nil {
			cancel()
			return
		}
		c.data <- v
	}
}
