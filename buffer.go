package main

import "io"

type buffer struct {
	bytes chan []byte
	data  []byte
}

func NewBuffer() *buffer {
	// todo: bytes可以做成带缓存的channel吗？
	return &buffer{bytes: make(chan []byte)}
}

func (b *buffer) Read(p []byte) (int, error) {
	ok := true
	for ok && len(b.data) == 0 {
		b.data, ok = <-b.bytes
	}
	if !ok || len(b.data) == 0 {
		return 0, io.EOF
	}

	l := copy(p, b.data)
	b.data = b.data[l:]
	return l, nil
}
