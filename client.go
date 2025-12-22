package memcached_go

import (
	"context"
	"errors"
	"memcached-go/gonet"
	"memcached-go/mmc"
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

func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	getMsg := mmc.NewGet(key)
	err := c.cli.Call(ctx, getMsg)
	if err != nil {
		return nil, err
	}
	if getMsg.Error != nil && errors.Is(getMsg.Error, mmc.ErrMiss) {
		return nil, nil
	}
	return getMsg.Value, nil
}

func (c *Client) Set(ctx context.Context, key string, val []byte) error {
	setMsg := mmc.NewSet(key, 0, val)
	err := c.cli.Call(ctx, setMsg)
	if err != nil {
		return err
	}
	if setMsg.Error != nil {
		return setMsg.Error
	}
	return nil
}
