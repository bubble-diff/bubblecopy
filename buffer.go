package main

import (
	"io"
)

type buffer struct {
	bytes chan []byte
	data  []byte
}

func NewBuffer() *buffer {
	// 这里，必须是无缓存的channel，因为channel是stream进行close的。
	// 如果带缓存，stream关掉channel后，consumer会消费失败。
	// todo: 将close交给consume去做？这样就可以带缓存了
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
