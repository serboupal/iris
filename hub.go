package iris

import (
	"context"
	"math/rand"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Hub struct {
	clients   map[uuid.UUID]*Peer
	clientsMu sync.RWMutex

	channels   map[string][]*Peer
	channelsMu sync.RWMutex

	services   map[string][]*Peer
	servicesMu sync.RWMutex

	OringinFunc func(r *http.Request) bool
	addr        string

	Exit chan error
}

type Statics struct {
	Name     string              `json:"name"`
	Total    int                 `json:"total_clients"`
	Clients  map[uuid.UUID]*Peer `json:"clients"`
	Services []string            `json:"services"`
	Channels []string            `json:"channels"`
	Version  string              `json:"version"`
}

func NewHub(addr string) *Hub {
	return &Hub{
		Exit: make(chan error),

		clients:  make(map[uuid.UUID]*Peer),
		services: make(map[string][]*Peer),
		channels: make(map[string][]*Peer),
		addr:     addr,
	}
}

func (h *Hub) Listen(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ErrContextDone
		case e := <-h.Exit:
			return e
		default:
			h.watchPeers()
		}
	}
}

func (h *Hub) ConnectPeer(w http.ResponseWriter, r *http.Request) (*Peer, error) {
	var err error
	c := Peer{
		Id:   uuid.New(),
		hub:  h,
		data: make(chan Cmd),
	}

	upgrader := websocket.Upgrader{}
	if h.OringinFunc != nil {
		upgrader.CheckOrigin = h.OringinFunc
	}

	c.conn, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	//websocket don't support this header but it can be added in proxy
	if r.Header.Get("X-Real-IP") != "" {
		c.Addr = r.Header.Get("X-Real-IP")
	} else {
		c.Addr = c.conn.LocalAddr().String()
	}

	h.clientsMu.Lock()
	h.clients[c.Id] = &c
	h.clientsMu.Unlock()

	go c.Fan(context.Background())
	return &c, nil
}

func (h *Hub) Statics() Statics {
	keys := make([]string, len(h.services))
	i := 0
	for k := range h.services {
		keys[i] = k
		i++
	}

	ckeys := make([]string, len(h.channels))
	i = 0
	for k := range h.channels {
		ckeys[i] = k
		i++
	}
	sr := Statics{
		Name:     "iris",
		Total:    len(h.clients),
		Clients:  h.clients,
		Services: keys,
		Channels: ckeys,
	}
	return sr
}

func (h *Hub) watchPeers() {
	h.clientsMu.Lock()
	for uid, d := range h.clients {
		peer := h.clients[uid]
		select {
		case m := <-d.data:
			// Process the message for the specific client
			switch m.Command {
			case Response:
				if m.ReplyTo != nil {
					if v, ok := h.clients[*m.ReplyTo]; ok {
						v.Send(m)
						break
					}
				}
				peer.disconnectMsg = ErrMalformedMsg

			case Subscribe:
				h.channelsMu.Lock()
				for _, s := range m.Data {
					h.channels[s] = append(h.channels[s], peer)
					peer.Channels = append(peer.Channels, s)
				}
				h.channelsMu.Unlock()

			case Publish:
				if len(m.Data) == 0 {
					peer.disconnectMsg = ErrMalformedMsg
					break
				}
				if ch, ok := h.channels[m.Data[0]]; ok {
					h.channelsMu.RLock()
					for _, p := range ch {
						m := NewCmd(m.Id, Publish, m.Data)
						p.Send(m)
					}
					h.channelsMu.RUnlock()
				}

			case Provide:
				h.servicesMu.Lock()
				for _, s := range m.Data {
					h.services[s] = append(h.services[s], peer)
					peer.Services = append(peer.Services, s)
				}
				h.servicesMu.Unlock()

			case Consume:
				m.ReplyTo = &uid
				if len(m.Data) == 0 {
					break
				}
				m.ReplyTo = &uid
				if v, ok := h.services[m.Data[0]]; len(v) > 0 && ok {
					h.servicesMu.RLock()
					provider := v[rand.Intn(len(v))]
					provider.Send(m)
					h.servicesMu.RUnlock()
					break
				}
				peer.SendError(m.Id, ErrServiceNotAvailable)

			default:
				peer.disconnectMsg = ErrInvalidCode
			}
		default:
		}
		if peer.disconnectMsg != nil {
			delete(h.clients, peer.Id)
			peer.SendError(nil, peer.disconnectMsg)
			peer.Disconnect()
		}
	}
	h.clientsMu.Unlock()
}
