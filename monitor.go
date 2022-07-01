package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

var (
	res   ParseInfoRes
	count int
)

type MonitorInfo struct {
	Timestamp   int64  `json:"timestamp,omitempty"`
	CpuInfo     string `json:"cpu,omitempty"`
	MemoryInfo  string `json:"mem,omitempty"`
	CpuTempInfo string `json:"temp,omitempty"`
	Serial      string `json:"serial,omitempty"`
	Sdk         int    `json:"sdkVersion,omitempty"`
	Cores       int    `json:"cpuCores,omitempty"`
	IP          string `json:"ip,omitempty"`
	Memory      int    `json:"totalMem,omitempty"`
}

type ParseInfo struct {
	IP           string `json:"ip,omitempty"`
	Serial       string `json:"serial,omitempty"`
	Cores        int    `json:"cpuCores,omitempty"`
	CPU          string `json:"cpu,omitempty"`
	Sdk          int    `json:"sdkVersion,omitempty"`
	ImageVersion string `json:"imageVersion,omitempty"`
	AppVersion   string `json:"appVersion,omitempty"`
	Memory       int    `json:"memory,omitempty"`
}

type ParseInfoRes struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    struct {
		URL string `json:"url,omitempty"`
	} `json:"data,omitempty"`
}

// Report http server
func reportServer(url string, content []byte, contentType string) (string, error) {
	// log.Infof("content info: %s", content)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(content))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", contentType)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New("push http error code: " + resp.Status)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body), nil
}

// cpu Memory temperature
func monitor(addr string) {
	log.Infof("Enter monitoring report [%s]\n", addr)
	deviceInfo := getDeviceInfo()
	job := cron.New()
	count = 1
	// 信息上报
	job.AddFunc("0 0/1 * * * ?", func() {
		outIp, _ := getOutboundIP()
		ip := outIp.String()
		if reflect.DeepEqual(res, ParseInfoRes{}) || count >= 5 {
			count = 1
			parseIPInfo(addr, ip)
		}
		if res.Data.URL == "" {
			count = 5
			log.Infof("[%s] 监控上报服务器地址为空", ip)
			return
		}
		count++
		memoryInfo, _ := parseAllMemoryInfo()
		cpuInfo, _ := parseAllTopCPUInfo()
		tempInfo, _ := parseCPUTempInfo()

		info := &MonitorInfo{
			Timestamp:   time.Now().Unix(),
			CpuInfo:     cpuInfo,
			MemoryInfo:  memoryInfo,
			CpuTempInfo: tempInfo,
			IP:          ip,
			Serial:      deviceInfo.Serial,
			Cores:       deviceInfo.Cpu.Cores,
			Sdk:         deviceInfo.Sdk,
			Memory:      deviceInfo.Memory.Total,
		}
		str, err := json.Marshal(info)
		if err != nil {
			log.Error(err)
		}
		monitor, err :=
			reportServer(res.Data.URL, str, "application/json")
		if err != nil {
			log.Infof("[%s] 监控信息上报失败 [%s]", ip, err.Error())
		} else {
			log.Infof("monitor report result [%s]", monitor)
		}

	})

	go job.Start()
}

// IP addr 上报
func parseIPInfo(addr string, ip string) error {
	log.Infof("IP地址上报 [%s] \n", addr+"/device/report")
	deviceInfo := getDeviceInfo()
	//reflect.ValueOf(deviceInfo).Elem().Field(8).SetString(ip)
	content, err := ioutil.ReadFile("/data/versions.txt")
	info := &ParseInfo{
		Serial:       deviceInfo.Serial,
		IP:           ip,
		Cores:        deviceInfo.Cpu.Cores,
		CPU:          getCachedProperty("ro.product.cpu.abi"),
		Sdk:          deviceInfo.Sdk,
		ImageVersion: getCachedProperty("ro.build.id"),
		AppVersion:   string(content),
		Memory:       deviceInfo.Memory.Total,
	}
	str, err := json.Marshal(info)
	if err != nil {
		return err
	}
	body, err := reportServer(addr+"/device/report", str, "application/json")
	log.Infof("IP地址上报返回Body [%s] \n", body)
	if err != nil {
		log.Infof("IP地址上报请求失败 [%s] \n", err.Error())
		return err
	}
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		log.Infof("IP地址上报失败 [%s] err [%s] \n", body, err.Error())
		return err
	}
	log.Infof("IP地址上报结果 [%d] \n", res.Code)
	return err
}

// parse all Memory Info
func parseAllMemoryInfo() (info string, err error) {
	output, err := Command{
		Args:    []string{"dumpsys", "meminfo", "--local"},
		Timeout: 60 * time.Second,
	}.CombinedOutputString()
	if err != nil {
		log.Error(err)
		return
	}
	info = output
	return
}

// parse all top CPU Info
func parseAllTopCPUInfo() (info string, err error) {
	cmd := []string{"top", "-b", "-n", "1", "-d", "1"}
	sdk, err := strconv.Atoi(getCachedProperty("ro.build.version.sdk"))
	// 兼容低版本 Android
	if sdk < 28 {
		cmd = append(cmd[:1], cmd[2:]...)
	}
	output, err := Command{
		Args:    cmd,
		Timeout: 10 * time.Second,
	}.CombinedOutputString()
	if err != nil {
		log.Error(err)
		return
	}
	info = output
	return
}

// parse CPU temperature Info
func parseCPUTempInfo() (info string, err error) {
	output, err := Command{
		Args:    []string{"cat", "/sys/class/thermal/thermal_zone0/temp"},
		Timeout: 10 * time.Second,
	}.CombinedOutputString()
	if err != nil {
		log.Error(err)
		return
	}

	if output == "" {
		err = errors.New("cat CPU temperature error")
		return
	}
	info = output
	return
}
