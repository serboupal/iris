package iris

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/exp/slices"
)

type Peer struct {
	Addr     string   `json:"addr"`
	Count    int      `json:"msg_count"`
	Services []string `json:"services,omitempty"`
	Channels []string `json:"channels,omitempty"`

	Id            uuid.UUID `json:"-"`
	hub           *Hub
	conn          *websocket.Conn
	data          chan Cmd
	disconnectMsg error
}

func (p *Peer) Fan() {
	for {
		m := Cmd{}
		err := p.conn.ReadJSON(&m)
		if err != nil {
			p.disconnectMsg = ErrConnectionClose
			p.data <- NewCmd(nil, Disconnect, nil)
			return
		}
		p.data <- m
	}
}

func (p *Peer) Send(m Cmd) error {
	p.Count++
	err := p.conn.WriteJSON(m)
	if err != nil {
		return err
	}
	return nil
}

func (p *Peer) SendError(err error) error {
	m := NewCmd(nil, Error, []string{err.Error()})
	return p.Send(m)

}

func (p *Peer) Disconnect() error {
	if len(p.Services) > 0 {
		p.hub.servicesMu.Lock()
		for _, v := range p.Services {
			p.hub.services[v] = slices.DeleteFunc(p.hub.services[v], func(e *Peer) bool {
				return e == p
			})
			if len(p.hub.services[v]) == 0 {
				delete(p.hub.services, v)
			}
		}

		p.hub.servicesMu.Unlock()
	}

	if len(p.Channels) > 0 {
		p.hub.channelsMu.Lock()
		for _, v := range p.Channels {
			p.hub.channels[v] = slices.DeleteFunc(p.hub.channels[v], func(e *Peer) bool {
				return e == p
			})
			if len(p.hub.channels[v]) == 0 {
				delete(p.hub.channels, v)
			}
		}

		p.hub.channelsMu.Unlock()
	}

	return p.conn.Close()
}
