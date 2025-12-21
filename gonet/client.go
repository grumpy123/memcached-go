package gonet

import (
	"context"
	"sync"
)

type Client struct {
	addr             string
	minCons, maxCons int

	slots chan *Connection

	conns    []*Connection
	connLock sync.Mutex
}

func NewClient(addr string, minCons, maxCons int) (*Client, error) {
	c := &Client{
		addr:    addr,
		minCons: minCons,
		maxCons: maxCons,

		slots: make(chan *Connection, maxCons),
		conns: make([]*Connection, maxCons),
	}
	err := c.connect(minCons)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) connect(n int) error {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	for i := 0; i < n; i++ {
		conn, err := NewConnection(c.addr)
		if err != nil {
			return err
		}
		c.conns[i] = conn
		c.slots <- conn
	}
	return nil
}

func (c *Client) Call(ctx context.Context, msg Message) error {
	select {
	case conn := <-c.slots:
		// todo: handle disconnected client and don't return it to the pool
		req, err := conn.Send(ctx, msg)
		// Request sent or canceled, returning the connection to the pool
		c.slots <- conn

		if err != nil {
			return err
		}

		select {
		case <-req.completed:
			return req.err
		case <-ctx.Done():
			return ctx.Err()
		}

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) Close() {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	for _, conn := range c.conns {
		conn.Close()
	}
	c.conns = nil
}
