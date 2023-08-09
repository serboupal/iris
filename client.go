package iris

import (
	"context"

	"github.com/gorilla/websocket"
)

type Client struct {
	Peer
	CmdHandler func(c *Client, cmd *Cmd)
}

func NewClient(addr string) *Client {
	return &Client{
		Peer: Peer{Addr: addr,
			data: make(chan Cmd)},
	}
}

func (c *Client) Listen(ctx context.Context) error {
	var err error
	c.conn, _, err = websocket.DefaultDialer.Dial(c.Addr, nil)
	if err != nil {
		return err
	}
	go c.Fan(ctx)

	for {
		select {
		case <-ctx.Done():
			return ErrContextDone
		case m := <-c.data:
			switch m.Command {
			case Disconnect:
				c.disconnectMsg = ErrDisconnect
			default:
				c.do(&m)
			}
		}
		if c.disconnectMsg != nil {
			c.Disconnect()
			return c.disconnectMsg
		}
	}
}

func (c *Client) do(cmd *Cmd) {
	if c.CmdHandler == nil {
		panic("CmdHandler not implemented")
	}
	c.CmdHandler(c, cmd)
}
