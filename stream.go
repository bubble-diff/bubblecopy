package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
	"github.com/sirupsen/logrus"
)

type tcpStream struct {
	factoryWg *sync.WaitGroup
	// protocol tcpStream挟带数据的上层协议类型
	protocol string
	// isDetect 已经确定该Stream的协议类型
	isDetect bool
	c2sBuf   *buffer
	s2cBuf   *buffer
}

func (s *tcpStream) Accept(tcp *layers.TCP, ci gopacket.CaptureInfo, dir reassembly.TCPFlowDirection, nextSeq reassembly.Sequence, start *bool, ac reassembly.AssemblerContext) bool {
	if *start {
		return true // Important! First SYN packet must be accepted.
	}
	// 当我们检测到应用层协议后，创建消费者进行消费。
	if !s.isDetect {
		s.protocol = guessProtocol(tcp.Payload)
		if s.protocol == UnknownType {
			return false // drop it.
		}
		s.isDetect = true
		s.factoryWg.Add(1)
		go s.consume()
	}
	return true
}

func (s *tcpStream) ReassembledSG(sg reassembly.ScatterGather, ac reassembly.AssemblerContext) {
	dir, _, _, _ := sg.Info()
	l, _ := sg.Lengths()
	data := sg.Fetch(l)
	if l > 0 {
		if dir == reassembly.TCPDirClientToServer {
			s.c2sBuf.bytes <- data
		} else {
			s.s2cBuf.bytes <- data
		}
	}
}

// ReassemblyComplete will be called when stream receive two endpoint FIN packet.
func (s *tcpStream) ReassemblyComplete(ac reassembly.AssemblerContext) bool {
	close(s.c2sBuf.bytes)
	close(s.s2cBuf.bytes)
	return true
}

// consume 消费两个缓存中的数据进行下一步处理
func (s *tcpStream) consume() {
	defer s.factoryWg.Done()

	switch s.protocol {
	case HttpType:
		handleHttp(s.c2sBuf, s.s2cBuf)
	}
}

func handleHttp(c2s, s2c io.Reader) {
	c2sReader := bufio.NewReader(c2s)
	s2cReader := bufio.NewReader(s2c)
	for {
		req, err := http.ReadRequest(c2sReader)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		} else if err != nil {
			log.Println(err)
			continue
		}
		resp, err := http.ReadResponse(s2cReader, nil)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		} else if err != nil {
			log.Println(err)
			continue
		}

		// todo: 我们目前只是将http req/resp以日志的形式打印下来
		//  后期我们在这里需要添加过滤http流量，以及转发至replayer的能力
		bytes, err := httputil.DumpRequest(req, true)
		if err != nil {
			log.Println(err)
		}
		req.Body.Close()
		logrus.Debug(string(bytes))

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
		}
		resp.Body.Close()
		logrus.Debug(string(body))
	}
}
