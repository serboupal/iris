package iris

import "github.com/google/uuid"

type Command uint8

const (
	Data Command = iota
	Error
	Subscribe
	Publish
	Unsubscribe
	Produce
	Consume
	Leave
)

type Msg struct {
	Id     *uuid.UUID `json:"id,omitempty"`
	Dest   *uuid.UUID `json:"dest,omitempty"`
	Source *uuid.UUID `json:"source,omitempty"`
	Cmd    Command    `json:"cmd,omitempty"`
	Data   []string   `json:"data,omitempty"`
}
