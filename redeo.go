package redeo

import (
	"errors"
)

var (
	errInvalidMultiBulkLength = errors.New("Protocol error: invalid multibulk length")
	errInvalidBulkLength      = errors.New("Protocol error: invalid bulk length")
)

// UnknownCommand returns an unknown command error string
func UnknownCommand(cmd string) string {
	return "ERR unknown command '" + cmd + "'"
}

// WrongNumberOfArgs returns an unknown command error string
func WrongNumberOfArgs(cmd string) string {
	return "ERR wrong number of arguments for '" + cmd + "' command"
}

// Handler is an abstract handler interface
type Handler interface {
	// ServeRedeo serves a request. If the ResponseBuffer remains empty
	// after the request, an inline "+OK\r\n" string will be returned
	// to the client by default.
	ServeRedeo(w *ResponseBuffer, c *Command)
}

// HandlerFunc is a callback function, implementing Handler.
type HandlerFunc func(w *ResponseBuffer, c *Command)

// ServeRedeo calls f(w, c).
func (f HandlerFunc) ServeRedeo(w *ResponseBuffer, c *Command) { f(w, c) }
