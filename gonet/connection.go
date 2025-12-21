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

type PendingMessage struct {
	msg       Message
	err       error
	completed chan struct{}
}

type Connection struct {
	conn net.Conn

	requests chan *PendingMessage
	pending  chan *PendingMessage
}

func NewConnection(server string) (*Connection, error) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return nil, err
	}

	c := &Connection{
		conn: conn,

		requests: make(chan *PendingMessage),       // no buffer, just synchronize writers and connection
		pending:  make(chan *PendingMessage, 1024), // todo: buffer here, in the future make it expand dynamically
	}
	go c.requestLoop()
	go c.responseLoop()
	return c, nil
}

func (c *Connection) Send(ctx context.Context, msg Message) (*PendingMessage, error) {
	req := &PendingMessage{msg: msg, completed: make(chan struct{})}
	select {
	case c.requests <- req:
		return req, nil
	case <-ctx.Done():
		// The message was never queued, we have to close the completed channel to avoid leaking it
		close(req.completed)
		return nil, ctx.Err()
	}
}

func (c *Connection) Call(ctx context.Context, msg Message) error {
	req, err := c.Send(ctx, msg)
	if err != nil {
		return err
	}

	select {
	case <-req.completed:
	case <-ctx.Done():
		return ctx.Err()
	}

	return req.err
}

// Close closes the client connection. Calls to Send or Call after Close will panic.
func (c *Connection) Close() {
	close(c.requests)
}

func (c *Connection) requestLoop() {
	defer close(c.pending)

	w := bufio.NewWriter(c.conn)
	for req := range c.requests {
		if w == nil {
			req.err = ErrConnClosed
			close(req.completed)
			continue
		}

		err := req.msg.WriteRequest(w)
		if err == nil {
			err = w.Flush()
		}

		if err != nil {
			// stop writing
			w = nil
			req.err = fmt.Errorf("sending error: %w", err)
			close(req.completed)
			continue
		}

		c.pending <- req
	}
}

func (c *Connection) responseLoop() {
	defer c.closeConnection()

	r := bufio.NewReader(c.conn)
	for resp := range c.pending {
		if r == nil {
			resp.err = ErrConnClosed
			close(resp.completed)
			continue
		}

		err := resp.msg.ReadResponse(r)
		if err != nil {
			resp.err = fmt.Errorf("receiving error: %w", err)
			// stop reading
			r = nil
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
