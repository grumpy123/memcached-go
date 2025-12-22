package mmc

import (
	"bufio"
	"fmt"
)

var (
	getCmd = []byte("get ")
)

type Get struct {
	// Request
	Key []byte

	// Response
	Flags uint16
	Value []byte
	Error error
}

func NewGet(key string) *Get {
	// todo: validate key
	return &Get{Key: []byte(key)}
}

func (g *Get) WriteRequest(w *bufio.Writer) error {
	_, err := w.Write(getCmd)
	if err != nil {
		return err
	}

	_, err = w.Write(g.Key)
	if err != nil {
		return err
	}

	_, err = w.Write(newLine)
	if err != nil {
		return err
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (g *Get) ReadResponse(r *bufio.Reader) error {
	header, err := respHeader(r)
	if err != nil {
		return fmt.Errorf("read response header: %w", err)
	}

	err = maybeError(header)
	if err != nil {
		g.Error = err
		return nil
	}

	if isEnd(header) {
		g.Error = ErrMiss
		return nil
	}

	flags, val, err := readValue(r, header, g.Key)
	if err != nil {
		return err
	}

	g.Flags = flags
	g.Value = val
	return nil
}
