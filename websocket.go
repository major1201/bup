package main

import (
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    4 * 4096,
	WriteBufferSize:   4 * 4096,
	EnableCompression: true,
}

type wsConnection struct {
	conn *websocket.Conn
	rBuf *bytes.Reader
}

func (c *wsConnection) Write(p []byte) (n int, err error) {
	writer, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	n = len(p)
	if !utf8.Valid(p) { // XXX: fix utf8 issue
		v := make([]rune, 0, len(p))
		for _, r := range string(p) {
			if r == utf8.RuneError {
				continue
			}
			v = append(v, r)
		}
		p = []byte(string(v))
	}
	defer writer.Close()
	_, err = writer.Write(p)
	return
}

func (c *wsConnection) Read(p []byte) (int, error) {
	for c.rBuf == nil || c.rBuf.Len() == 0 {
		mt, b, err := c.conn.ReadMessage()
		if err != nil {
			return 0, err
		}
		if len(b) == 0 {
			continue
		}
		if mt != websocket.TextMessage {
			continue
		}
		c.rBuf = bytes.NewReader(b)
	}
	n, err := c.rBuf.Read(p)
	if err == io.EOF {
		err = nil
	}
	return n, err
}

func (c *wsConnection) Close() error {
	return c.conn.Close()
}
