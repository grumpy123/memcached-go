package mmc

import (
	"bufio"
	"bytes"
	"fmt"
	"time"
)

var (
	setCmd = []byte("set ")
)

type Set struct {
	// Request
	Key     []byte
	Flags   uint16
	Value   []byte
	Exptime int32

	// Response
	Error error
}

func NewSet(key string, flags uint16, value []byte, ttl time.Duration) *Set {
	// todo: validate key
	return &Set{Key: []byte(key), Flags: flags, Value: value, Exptime: ttlToExptime(ttl)}
}

func (s *Set) WriteRequest(w *bufio.Writer) error {
	_, err := w.Write(setCmd)
	if err != nil {
		return err
	}

	_, err = w.Write(s.Key)
	if err != nil {
		return err
	}

	// todo: support duration
	params := fmt.Sprintf(" %d %d %d\r\n", s.Flags, 0, len(s.Value))
	_, err = w.WriteString(params)
	if err != nil {
		return err
	}

	_, err = w.Write(s.Value)
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

func (s *Set) ReadResponse(r *bufio.Reader) error {
	header, err := respHeader(r)
	if err != nil {
		return err
	}

	err = maybeError(header)
	if err != nil {
		s.Error = err
		return nil
	}

	if !bytes.Equal(header[0], stored) {
		return fmt.Errorf("expected stored, but got %q: %w", string(header[0]), ErrBadResponse)
	}
	return nil
}
