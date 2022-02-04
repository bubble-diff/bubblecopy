package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/reassembly"
	"github.com/sirupsen/logrus"
)

var debugmode bool

func init() {
	flag.BoolVar(&debugmode, "debug", false, "Run as debug mode, read settings file to override task configuration if existsed.")
	flag.Parse()

	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02 15:04:05",
		CallerPrettyfier: callerPrettyfier,
	})
	if debugmode {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetReportCaller(true)
		logrus.Infof("<---------debug mode--------->")
	}

	configuration.init()
}

func main() {
	handle, err := pcap.OpenLive(configuration.Device, SnapshotLen, false, pcap.BlockForever)
	defer handle.Close()
	if err != nil {
		logrus.Error(err)
		logrus.Fatal("Try sudo.")
	}

	// 过滤出当前服务的流量
	filter := fmt.Sprintf(
		"(src port %s and src host %s) or (dst port %s and dst host %s)",
		configuration.Port, configuration.DeviceIPv4,
		configuration.Port, configuration.DeviceIPv4,
	)
	logrus.Debugf("Set bpf filter as: %s", filter)
	if err := handle.SetBPFFilter(filter); err != nil {
		logrus.Fatal(err)
	}

	source := gopacket.NewPacketSource(handle, handle.LinkType())
	source.NoCopy = true

	streamFactory := &tcpStreamFactory{}
	streamPool := reassembly.NewStreamPool(streamFactory)
	assembler := reassembly.NewAssembler(streamPool)

	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	defer streamFactory.WaitConsumers()
	for {
		// todo: 等待Diff任务启动，若未启动，请勿进行抓包消耗CPU
		// your code here...

		select {
		case <-ticker.C:
			// 停止监听30秒内无数据传输的连接
			assembler.FlushCloseOlderThan(time.Now().Add(time.Second * -30))
		case packet := <-source.Packets():
			tcp := packet.Layer(layers.LayerTypeTCP)
			if tcp != nil {
				tcp := tcp.(*layers.TCP)
				assembler.Assemble(packet.NetworkLayer().NetworkFlow(), tcp)
			}
		}
	}
}
