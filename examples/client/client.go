package main

import (
	"context"

	"github.com/serboupal/iris"
)

const defHost = "localhost:8888"

func main() {
	const endpoint = "ws://" + defHost + "/helo"
	c := iris.NewClient(endpoint)
	c.CmdHandler = do

	panic(c.Listen(context.Background()))
}

func do(c *iris.Client, cmd *iris.Cmd) {
	switch cmd.Command {
	case iris.Hello:
		// add providers
		m := iris.NewCmd(nil, iris.Provide, []string{"test"})
		c.Send(m)
	case iris.Consume:
		if len(cmd.Data) < 1 || cmd.ReplyTo == nil {
			return
		}
		m := iris.NewCmd(cmd.Id, iris.Response, []string{"testReply"})
		m.ReplyTo = cmd.ReplyTo
		c.Send(m)
	default:

	}
}
