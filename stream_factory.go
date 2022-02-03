package main

import (
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
)

// tcpStreamFactory 创建新的 tcpStream，并创建消费者消费数据
type tcpStreamFactory struct {
	wg sync.WaitGroup
}

func (f *tcpStreamFactory) New(netFlow, tcpFlow gopacket.Flow, tcp *layers.TCP, ac reassembly.AssemblerContext) reassembly.Stream {
	s := &tcpStream{}
	s.c2sBuf = &buffer{
		bytes: make(chan []byte),
	}
	s.s2cBuf = &buffer{
		bytes: make(chan []byte),
	}
	f.wg.Add(1)
	go s.consume()
	return s
}

func (f *tcpStreamFactory) WaitConsumers() {
	f.wg.Wait()
}
