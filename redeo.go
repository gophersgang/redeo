package redeo

import (
	"github.com/bsm/redeo/resp"
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
	ServeRedeo(w resp.ResponseWriter, c *resp.Command)
}

// HandlerFunc is a callback function, implementing Handler.
type HandlerFunc func(w resp.ResponseWriter, c *resp.Command)

// ServeRedeo calls f(w, c).
func (f HandlerFunc) ServeRedeo(w resp.ResponseWriter, c *resp.Command) { f(w, c) }
