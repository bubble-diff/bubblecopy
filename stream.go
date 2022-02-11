package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"github.com/bubble-diff/bubblecopy/pb"
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
	// false to receive last ack for avoiding New tcpStream.
	return false
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
		// read http request and response.
		req, err := http.ReadRequest(c2sReader)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		} else if err != nil {
			logrus.Error(err)
			continue
		}
		resp, err := http.ReadResponse(s2cReader, nil)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		} else if err != nil {
			logrus.Error(err)
			continue
		}

		// send http req/resp to bubblereplay.
		err = sendReqResp(req, resp)
		if err != nil {
			logrus.Errorf("send old req/resp failed, %s", err)
		} else {
			logrus.Info("send old req/resp ok")
		}

		req.Body.Close()
		resp.Body.Close()
	}
}

// sendReqResp 将old req/resp发送至replay服务进行进一步处理
// todo: 这个函数应该是协议无关的，现在参数为http协议。
func sendReqResp(req *http.Request, resp *http.Response) (err error) {
	// 序列化req/resp
	rawReq, err := httputil.DumpRequest(req, true)
	if err != nil {
		return err
	}
	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// send them
	request := &pb.AddRecordReq{
		Record: &pb.Record{
			TaskId:  configuration.Taskid,
			OldReq:  rawReq,
			OldResp: rawResp,
			NewResp: nil,
		},
	}
	rawpb, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	api := fmt.Sprintf("http://%s%s", configuration.ReplaySvrAddr, ApiAddRecord)
	apiResp, err := http.Post(api, "application/octet-stream", bytes.NewReader(rawpb))
	if err != nil {
		return err
	}

	// parse api response
	var response pb.AddRecordResp
	rawApiResp, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(rawApiResp, &response)
	if err != nil {
		return err
	} else if response.Code != 0 {
		return errors.New(response.Msg)
	}

	return apiResp.Body.Close()
}
