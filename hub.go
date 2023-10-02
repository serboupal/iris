package iris

import (
	"context"
	"math/rand"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Hub struct {
	OringinFunc func(r *http.Request) bool
	DoFunc      func(p *Peer, c Msg)

	addr       string
	writeWait  time.Duration
	pingWait   time.Duration
	pingPeriod time.Duration

	clients map[uuid.UUID]*Peer
	groups  map[string]*Group

	disconnect chan *Peer
	connect    chan *Peer
	ping       chan *Peer

	data chan *dataReq
	send chan *dataReq
}

type groupReq struct {
	name    string
	service bool
	peer    *Peer
}

type dataReq struct {
	msg  Msg
	peer *Peer
}

type Group struct {
	Queue bool    `json:"queue"`
	Peers []*Peer `json:"-"`
	Count int     `json:"count"`
}

type Statics struct {
	Name    string              `json:"name"`
	Total   int                 `json:"total_clients"`
	Clients map[uuid.UUID]*Peer `json:"clients"`
	Groups  map[string]*Group   `json:"groups"`
	Version string              `json:"version"`
}

func NewHub(addr string, writeWait time.Duration, pongWait time.Duration) *Hub {
	return &Hub{
		//store
		clients: make(map[uuid.UUID]*Peer),
		groups:  make(map[string]*Group),

		disconnect: make(chan *Peer),
		connect:    make(chan *Peer),
		ping:       make(chan *Peer),
		data:       make(chan *dataReq),
		send:       make(chan *dataReq, 100),

		addr:       addr,
		pingWait:   pongWait,
		writeWait:  writeWait,
		pingPeriod: (pongWait * 9) / 10,
	}
}

func (h *Hub) Listen(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ErrContextDone
		case p := <-h.connect:
			err := h.addClient(p)
			if err != nil {
				return err
			}

		case p := <-h.disconnect:
			err := h.removeClient(p)
			if err != nil {
				return err
			}

		case p := <-h.ping:
			err := h.pingClient(p)
			if err != nil {
				h.disconnect <- p
			}

		case d := <-h.data:
			h.processData(d)

		case d := <-h.send:
			err := d.peer.sendConn(d.msg, h.writeWait)
			if err != nil {
				h.disconnect <- d.peer
			}
		}
	}
}

func (h *Hub) ConnectPeer(w http.ResponseWriter, r *http.Request) (*Peer, error) {
	var err error
	p := Peer{
		Id:   uuid.New(),
		hub:  h,
		data: make(chan Msg),
	}

	upgrader := websocket.Upgrader{}
	if h.OringinFunc != nil {
		upgrader.CheckOrigin = h.OringinFunc
	}

	p.conn, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	//websocket don't support this header but it can be added in proxy
	if r.Header.Get("X-Real-IP") != "" {
		p.Addr = r.Header.Get("X-Real-IP")
	} else {
		p.Addr = p.conn.LocalAddr().String()
	}

	h.connect <- &p
	return &p, nil
}

func (h *Hub) Statics(serviceName string) Statics {
	sr := Statics{
		Name:    serviceName,
		Total:   len(h.clients),
		Clients: h.clients,
		Groups:  h.groups,
	}
	for k := range h.groups {
		h.groups[k].Count = len(h.groups[k].Peers)
	}
	return sr
}

func (h *Hub) Send(p *Peer, v Msg) {
	h.send <- &dataReq{
		peer: p,
		msg:  v,
	}
}

func (h *Hub) processData(d *dataReq) {
	if len(d.msg.Data) == 0 {
		return
	}

	if d.msg.Id == nil {
		id := uuid.New()
		d.msg.Id = &id
	}

	d.msg.Source = &d.peer.Id

	if d.msg.Dest != nil {
		if p, ok := h.clients[*d.msg.Dest]; ok {
			d.msg.Dest = nil
			p.Send(d.msg)
		}
		return
	}

	switch d.msg.Cmd {
	case Subscribe:
		g := groupReq{
			peer: d.peer,
			name: d.msg.Data[0],
		}
		h.join(&g)

	case Consume:
		g := groupReq{
			peer:    d.peer,
			name:    d.msg.Data[0],
			service: true,
		}
		h.join(&g)

	case Unsubscribe:
		g := groupReq{
			peer: d.peer,
			name: d.msg.Data[0],
		}
		h.leave(&g)

	case Publish:
		if _, ok := h.groups[d.msg.Data[0]]; !ok || h.groups[d.msg.Data[0]].Queue {
			break
		}
		for _, p := range h.groups[d.msg.Data[0]].Peers {
			p.Send(d.msg)
		}

	case Produce:
		if _, ok := h.groups[d.msg.Data[0]]; !ok || !h.groups[d.msg.Data[0]].Queue {
			break
		}
		r := rand.Int() % len(h.groups[d.msg.Data[0]].Peers)
		p := h.groups[d.msg.Data[0]].Peers[r]
		p.Send(d.msg)
	}

	if h.DoFunc != nil {
		go h.DoFunc(d.peer, d.msg)
	}
}

func (h *Hub) pingClient(p *Peer) error {
	err := p.conn.SetWriteDeadline(time.Now().Add(p.hub.writeWait))
	if err != nil {
		return err
	}
	err = p.conn.WriteMessage(websocket.PingMessage, nil)
	if err != nil {
		return err
	}
	return nil
}

func (h *Hub) addClient(p *Peer) error {
	h.clients[p.Id] = p
	go p.fan(context.Background())
	return nil
}

func (h *Hub) removeClient(p *Peer) error {
	delete(h.clients, p.Id)
	for k := range p.hub.groups {
		h.leave(&groupReq{name: k, peer: p})
	}
	return p.conn.Close()
}

func (h *Hub) join(g *groupReq) error {
	if slices.Contains(g.peer.Groups, g.name) {
		return nil
	}
	if _, ok := h.groups[g.name]; !ok {
		new := Group{
			Queue: g.service,
			Peers: []*Peer{},
		}
		h.groups[g.name] = &new
	}
	h.groups[g.name].Peers = append(h.groups[g.name].Peers, g.peer)
	g.peer.Groups = append(g.peer.Groups, g.name)
	return nil
}

func (h *Hub) leave(g *groupReq) error {
	if _, ok := h.groups[g.name]; !ok {
		return ErrInternal
	}
	h.groups[g.name].Peers = slices.DeleteFunc(h.groups[g.name].Peers, func(e *Peer) bool {
		return e == g.peer
	})
	if len(h.groups[g.name].Peers) == 0 {
		delete(h.groups, g.name)
	}
	return nil
}
