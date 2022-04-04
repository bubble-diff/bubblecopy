package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

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

	mu            sync.Mutex
	isTaskRunning bool
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

	// 启动后台去执行一些自动更新的动作
	go func() {
		logrus.Info("configuration background started.")
		for {
			c.setIsTaskRunning()
			c.reportDepolyed()
			time.Sleep(5 * time.Second)
		}
	}()

	logrus.WithField(
		"configuration",
		fmt.Sprintf("%+v", configuration),
	).Debug("configuration initialized.")
}

func (c *config) getIsTaskRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isTaskRunning
}

func (c *config) setIsTaskRunning() {
	var err error
	var apiResp struct {
		Err       string `json:"err"`
		IsRunning bool   `json:"is_running"`
	}

	logrus.Infof("[setIsTaskRunning] updating TaskID=%d status...", c.Taskid)
	// call bubblereplay
	api := fmt.Sprintf("http://%s%s/%d", c.ReplaySvrAddr, ApiTaskStatus, c.Taskid)
	resp, err := http.Get(api)
	if err != nil {
		logrus.Errorf("[setIsTaskRunning] call api failed, %s", err)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		logrus.Errorf("[setIsTaskRunning] decode json response failed, %s", err)
		return
	}
	resp.Body.Close()

	if apiResp.Err != "" {
		logrus.Errorf("[setIsTaskRunning] response return error, %s", err)
		return
	}

	// update state
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isTaskRunning = apiResp.IsRunning
}

func (c *config) reportDepolyed() {
	var err error
	var apiBody = struct {
		TaskID int64 `json:"task_id"`
		// Addr 基准服务地址
		Addr string `json:"addr"`
	}{
		TaskID: c.Taskid,
		Addr:   fmt.Sprintf("%s:%s", c.DeviceIPv4, c.Port),
	}
	var apiResp struct {
		Err string `json:"err"`
	}

	logrus.Infof("[reportDepolyed] reporting TaskID=%d has been deployed the bubblecopy...", c.Taskid)
	// call bubblereplay
	api := fmt.Sprintf("http://%s%s", c.ReplaySvrAddr, ApiSetDeployed)
	data, err := json.Marshal(apiBody)
	resp, err := http.Post(api, "application/json", bytes.NewReader(data))
	if err != nil {
		logrus.Errorf("[reportDepolyed] call api failed, %s", err)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		logrus.Errorf("[reportDepolyed] decode json response failed, %s", err)
		return
	}
	resp.Body.Close()

	if apiResp.Err != "" {
		logrus.Errorf("[reportDepolyed] response return error, %s", err)
		return
	}
}
