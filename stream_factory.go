package main

import (
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
)

// tcpStreamFactory 创建新的 tcpStream
type tcpStreamFactory struct {
	wg sync.WaitGroup
}

func (f *tcpStreamFactory) New(netFlow, tcpFlow gopacket.Flow, tcp *layers.TCP, ac reassembly.AssemblerContext) reassembly.Stream {
	s := &tcpStream{
		factoryWg: &f.wg,
		isDetect:  false,
		c2sBuf:    NewBuffer(),
		s2cBuf:    NewBuffer(),
	}
	return s
}

func (f *tcpStreamFactory) WaitConsumers() {
	f.wg.Wait()
}
