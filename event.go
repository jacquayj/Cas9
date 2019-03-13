package cas9

import "syscall/js"

type EventObj struct {
	js.Value
}

type EventHandler func(EventObj)

type SelectorEvents map[string]Event

type Event struct {
	On      string
	Handler func(EventObj)
}
