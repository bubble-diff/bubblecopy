package main

import (
	"errors"
	"fmt"
	"path"
	"runtime"
	"strings"

	"github.com/google/gopacket/pcap"
)

// getDeviceIpv4 获取网卡的ipv4地址，如果有的话。
func getDeviceIpv4(deviceName string) (ipv4 string, err error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return "", err
	}

	for _, device := range devices {
		if device.Name == deviceName {
			for _, address := range device.Addresses {
				if strings.IndexByte(address.IP.String(), '.') != -1 {
					return address.IP.String(), nil
				}
			}
			return "", errors.New(deviceName + ": no ipv4 for this interface")
		}
	}
	return "", errors.New(deviceName + ": no such device")
}

// callerPrettyfier 只显示文件名和行号。
func callerPrettyfier(frame *runtime.Frame) (function string, file string) {
	fileName := path.Base(frame.File)
	return "", fmt.Sprintf("%s:%d", fileName, frame.Line)
}
