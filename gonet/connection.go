package gonet

import (
	"bufio"
	"context"
	"fmt"
	"net"
)

type Message interface {
	WriteRequest(w *bufio.Writer) error
	ReadResponse(r *bufio.Reader) error
}

type pendingMessage struct {
	msg       Message
	err       error
	completed chan struct{}
}

type Connection struct {
	conn net.Conn

	requests chan *pendingMessage
	pending  chan *pendingMessage
}

func NewConnection(server string) (*Connection, error) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return nil, err
	}

	c := &Connection{
		conn: conn,

		requests: make(chan *pendingMessage),
		pending:  make(chan *pendingMessage),
	}
	go c.requestLoop()
	go c.responseLoop()
	return c, nil
}

func (c *Connection) Call(ctx context.Context, msg Message) error {
	req := &pendingMessage{msg: msg, completed: make(chan struct{})}
	select {
	case c.requests <- req:
	case <-ctx.Done():
		// The message was never queued, we have to close the completed channel to avoid leaking it
		close(req.completed)
		return ctx.Err()
	}

	select {
	case <-req.completed:
	case <-ctx.Done():
		return ctx.Err()
	}

	return req.err
}

// Close closes the client connection. Calls to Call after Close will panic.
func (c *Connection) Close() {
	close(c.requests)
}

func (c *Connection) requestLoop() {
	defer close(c.pending)

	w := bufio.NewWriter(c.conn)
	for req := range c.requests {
		err := req.msg.WriteRequest(w)
		if err == nil {
			err = w.Flush()
		}

		if err != nil {
			req.err = fmt.Errorf("sending error: %w", err)
			close(req.completed)
			// todo: drain remaining requests and fulfill with errors
			return
		}

		c.pending <- req
	}
}

func (c *Connection) responseLoop() {
	defer c.closeConnection()

	r := bufio.NewReader(c.conn)
	var err error
	for resp := range c.pending {
		if err == nil {
			err = resp.msg.ReadResponse(r)
		}
		if err != nil {
			resp.err = fmt.Errorf("receiving error: %w", err)
		}
		// assign the error to the response here?
		close(resp.completed)
	}
}

func (c *Connection) closeConnection() {
	if err := c.conn.Close(); err != nil {
		// todo: log on error instead of panicking
		panic(err)
	}
}
