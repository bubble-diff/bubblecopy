package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type config struct {
	// Taskid Diff任务ID
	Taskid int64 `json:"taskid"`
	// Secret 访问对应id任务配置的密钥
	Secret string `json:"secret"`
	// Device 网卡名称
	Device string `json:"interface"`
	// Port 被测服务端口
	Port string `json:"service_port"`
	// ReplaySvrAddr bubblereplay服务地址
	ReplaySvrAddr string `json:"replay_svr_addr"`

	// DeviceIPv4 网卡ipv4地址
	DeviceIPv4 string
}

var configuration = config{}

func (c *config) init() {
	bytes, err := os.ReadFile(SettingsFilePath)
	if err != nil {
		logrus.Error(err)
		logrus.Fatal("Need settings.json to get configuration.")
	}
	err = json.Unmarshal(bytes, &configuration)
	if err != nil {
		logrus.Fatal(err)
	}

	if configuration.ReplaySvrAddr == "" {
		logrus.Fatal("bubblereplay server addr not set")
	}

	c.DeviceIPv4, err = getDeviceIpv4(c.Device)
	if err != nil {
		logrus.Error(err)
		logrus.Fatalf("%s: this device has no ipv4 address.", c.Device)
	}

	logrus.WithField(
		"configuration",
		fmt.Sprintf("%+v", configuration),
	).Debug("configuration initialized.")
}
