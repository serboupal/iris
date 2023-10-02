package iris

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Peer struct {
	Addr   string   `json:"addr"`
	Count  int      `json:"msg_count"`
	Groups []string `json:"groups,omitempty"`

	Id     uuid.UUID `json:"-"`
	hub    *Hub
	conn   *websocket.Conn
	cancel context.CancelFunc
	data   chan Msg
}

func (p *Peer) fan(ctx context.Context) {
	ticker := time.NewTicker(p.hub.pingPeriod)
	defer ticker.Stop()
	ctx, p.cancel = context.WithCancel(ctx)
	go p.read(p.cancel)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.hub.ping <- p
		}
	}
}

func (p *Peer) read(cancel context.CancelFunc) {
	err := p.conn.SetReadDeadline(time.Now().Add(p.hub.pingWait))
	if err != nil {
		p.hub.disconnect <- p
		cancel()
		return
	}

	p.conn.SetPongHandler(func(string) error {
		p.conn.SetReadDeadline(time.Now().Add(p.hub.pingWait))
		return nil
	})

	for {
		v := Msg{}
		err := p.conn.ReadJSON(&v)
		if err != nil {
			p.hub.disconnect <- p
			cancel()
			return
		}
		e := dataReq{
			peer: p,
			msg:  v,
		}
		p.hub.data <- &e
	}
}

func (p *Peer) sendConn(v Msg, writeWait time.Duration) error {
	err := p.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		return err
	}
	err = p.conn.WriteJSON(v)
	if err != nil {
		return err
	}
	return nil
}

func (p *Peer) Send(v Msg) {
	r := dataReq{
		peer: p,
		msg:  v,
	}
	p.Count++
	p.hub.send <- &r
}
