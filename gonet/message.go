package gonet

import "bufio"

// Message represents a request and response of a protocol.
// If any of the methods returns an error the connection will be closed, as it's likely to be in dirty state.
// Any valid protocol-level errors must be encoded as part of the response, not returned as errors from  Message methods.
type Message interface {
	WriteRequest(w *bufio.Writer) error
	ReadResponse(r *bufio.Reader) error
}

// PendingMessage represents a Message wile being processed by the Client.
type PendingMessage struct {
	msg Message

	// Client-level error, most commonly ErrConnClosed.
	err error

	// Future, triggered when the response has been fully read, or error occurred.
	completed chan struct{}
}
