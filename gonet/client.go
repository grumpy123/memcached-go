package gonet

import (
	"context"
	"slices"
	"sync"
	"time"
)

type Client struct {
	addr    string
	minCons int
	maxCons int

	isOpen bool

	slots chan *Connection

	conns    []*Connection
	connLock sync.Mutex
}

func NewClient(addr string, minCons, maxCons int) (*Client, error) {
	c := &Client{
		addr:    addr,
		minCons: minCons,
		maxCons: maxCons,

		isOpen: true,

		slots: make(chan *Connection, maxCons),
		conns: make([]*Connection, 0, maxCons),
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
		c.conns = append(c.conns, conn)
		c.slots <- conn
	}
	return nil
}

func (c *Client) Call(ctx context.Context, msg Message) error {
	for {
		if len(c.slots) == 0 {
			go c.maybeGrow(0)
		}

		select {
		case conn := <-c.slots:
			if !conn.IsOpen() {
				// todo: also check if the pool still has minimum connections
				continue
			}

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
}

func (c *Client) maybeGrow(initialWait time.Duration) {
	if !c.isOpen {
		return
	}

	if initialWait > 0 {
		time.Sleep(initialWait)
	}

	if !c.connLock.TryLock() {
		// Already being handled
		return
	}
	defer c.connLock.Unlock()

	// Remove all dead connections first
	c.conns = slices.DeleteFunc(c.conns, func(conn *Connection) bool { return !conn.IsOpen() })

	if len(c.conns) < c.maxCons {
		conn, err := NewConnection(c.addr)
		if err != nil {
			// todo: real exponential backoff, randomized and with cap
			go c.maybeGrow(time.Second + initialWait*3/2) // guaranteed to be growing
			return
		}
		c.conns = append(c.conns, conn)
		c.slots <- conn
	}
}

func (c *Client) Close() {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	c.isOpen = false
	for _, conn := range c.conns {
		conn.Close()
	}
	c.conns = nil
}
