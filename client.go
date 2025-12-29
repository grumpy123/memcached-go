package memcached_go

import (
	"context"
	"errors"
	"memcached-go/gonet"
	"memcached-go/mmc"
	"time"
)

type Client struct {
	cli *gonet.Client
}

func NewClient(addr string, minConns, maxConns int) (*Client, error) {
	cli, err := gonet.NewClient(addr, minConns, maxConns)
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() {
	c.cli.Close()
}

func (c *Client) Get(ctx context.Context, key string) ([]byte, uint16, error) {
	getMsg := mmc.NewGet(key)
	err := c.cli.Call(ctx, getMsg)
	if err != nil {
		return nil, 0, err
	}
	if getMsg.Error != nil && errors.Is(getMsg.Error, mmc.ErrMiss) {
		return nil, 0, nil
	}
	return getMsg.Value, getMsg.Flags, nil
}

func (c *Client) GetV(ctx context.Context, key string) ([]byte, error) {
	val, _, err := c.Get(ctx, key)
	return val, err
}

func (c *Client) Set(ctx context.Context, key string, flags uint16, val []byte, ttl time.Duration) error {
	setMsg := mmc.NewSet(key, flags, val, ttl)
	err := c.cli.Call(ctx, setMsg)
	if err != nil {
		return err
	}
	if setMsg.Error != nil {
		return setMsg.Error
	}
	return nil
}

func (c *Client) SetV(ctx context.Context, key string, val []byte) error {
	return c.Set(ctx, key, 0, val, 0)
}
