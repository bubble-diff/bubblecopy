package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
	"github.com/sirupsen/logrus"
)

type tcpStream struct {
	c2sBuf *buffer
	s2cBuf *buffer
}

func (s *tcpStream) Accept(tcp *layers.TCP, ci gopacket.CaptureInfo, dir reassembly.TCPFlowDirection, nextSeq reassembly.Sequence, start *bool, ac reassembly.AssemblerContext) bool {
	// todo: 我们可以在这里检测tcp挟带的应用层数据
	//  Your code here...
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

func (s *tcpStream) ReassemblyComplete(ac reassembly.AssemblerContext) bool {
	close(s.c2sBuf.bytes)
	close(s.s2cBuf.bytes)
	// do not remove the connection to allow last ACK
	return false
}

// consume 消费两个缓存中的数据进行下一步处理
func (s *tcpStream) consume() {
	c2sReader := bufio.NewReader(s.c2sBuf)
	s2cReader := bufio.NewReader(s.s2cBuf)
	// todo: 这里等待stream检测出应用层类型后才能开始正确消费流量
	//  如你所见，目前默认为http数据，以后我们还想支持grpc的http2和thrift，kafka，redis...

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
