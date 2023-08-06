package iris

import (
	"github.com/google/uuid"
)

type Command int

const (
	Disconnect Command = iota
	Hello
	Error
	Subscribe
	Publish
	Provide
	Consume
	Response
)

type Cmd struct {
	Id      *uuid.UUID `json:"id"`
	ReplyTo *uuid.UUID `json:"reply,omitempty"`
	Command Command    `json:"cmd"`
	Data    []string   `json:"data,omitempty"`
}

func NewCmd(id *uuid.UUID, c Command, data []string) Cmd {
	if id == nil {
		e := uuid.New()
		id = &e
	}
	return Cmd{Id: id, Command: c, Data: data}
}
