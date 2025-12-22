package mmc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var (
	newLine = []byte("\r\n")
	space   = []byte(" ")

	end        = []byte("END")
	en         = []byte("EN")
	endOfValue = []byte("\r\nEND\r\n")

	value  = []byte("VALUE")
	stored = []byte("STORED")

	genError    = []byte("ERROR")
	clientError = []byte("CLIENT_ERROR")
	serverError = []byte("SERVER_ERROR")

	ErrClientError = errors.New("memcached client error")
	ErrServerError = errors.New("memcached server error")
	ErrGenError    = errors.New("memcached error")
	ErrBadResponse = errors.New("bad memcached response")

	ErrMiss = errors.New("miss")
)

func respHeader(r *bufio.Reader) ([][]byte, error) {
	// todo: consider ensure enough buffer always available? Premature optimization for realz :)
	bin, err := r.ReadSlice('\n')
	if err != nil {
		return nil, err
	}
	bin = dropTrailingNewLine(bin)
	parts := bytes.SplitN(bin, space, 2)
	return parts, nil
}

func dropTrailingNewLine(in []byte) []byte {
	// todo: error handling
	return in[:len(in)-2]
}

func maybeError(header [][]byte) error {
	// todo: consider terminating the connection after client error
	if err := maybeClientError(header); err != nil {
		return err
	}
	if err := maybeServerError(header); err != nil {
		return err
	}
	if err := maybeGenError(header); err != nil {
		return err
	}
	return nil
}

func maybeClientError(header [][]byte) error {
	return parseErrorX(header, clientError, ErrClientError)
}

func maybeServerError(header [][]byte) error {
	return parseErrorX(header, serverError, ErrServerError)
}

func maybeGenError(header [][]byte) error {
	return parseErrorX(header, genError, ErrGenError)
}

func parseErrorX(header [][]byte, errorX []byte, err error) error {
	if bytes.Equal(header[0], errorX) {
		if len(header) >= 2 {
			return fmt.Errorf("%w: %s", err, string(header[1]))
		}
		return err
	}
	return nil
}

func isEnd(header [][]byte) bool {
	return bytes.Equal(header[0], end)
}

func readValue(r *bufio.Reader, header [][]byte, key []byte) (uint16, []byte, error) {
	if !bytes.Equal(header[0], value) {
		return 0, nil, fmt.Errorf("expected value, but memcached returned %s: %w", string(header[0]), ErrBadResponse)
	}
	if len(header) < 2 {
		return 0, nil, fmt.Errorf("expected 3 more parts after value: %w", ErrBadResponse)
	}
	params := bytes.Split(header[1], space)
	if len(params) != 3 {
		return 0, nil, fmt.Errorf("expected 3 more parts after value, got %q: %w", string(header[1]), ErrBadResponse)
	}
	if !bytes.Equal(params[0], key) {
		return 0, nil, fmt.Errorf("incorrect key %q, requested %q: %w", string(params[0]), string(key), ErrBadResponse)
	}
	flags, err := strconv.ParseUint(string(params[1]), 10, 16)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid flags: %w", ErrBadResponse)
	}
	length, err := strconv.ParseUint(string(params[2]), 10, 32)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid length: %w", ErrBadResponse)
	}
	val := make([]byte, length)
	_, err = io.ReadFull(r, val)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read value: %w", ErrBadResponse)
	}

	ending := make([]byte, len(endOfValue))
	_, err = io.ReadFull(r, ending)
	if err != nil {
		return 0, nil, fmt.Errorf("after value expected \\r\\nEND\\r\\n: %w", ErrBadResponse)
	}
	if !bytes.Equal(ending, endOfValue) {
		return 0, nil, fmt.Errorf("after value expected \\r\\nEND\\r\\n, got %q: %w", string(ending), ErrBadResponse)
	}
	return uint16(flags), val, nil
}
