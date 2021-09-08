package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

type MonitorInfo struct {
	Uid                    string                `json:"uid,omitempty"`
	Serial                 string                `json:"serial,omitempty"`
	Timestamp              int64                 `json:"timestamp,omitempty"`
	Data                   string                `json:"data,omitempty"`
	Sdk                    int                   `json:"sdk,omitempty"`
	CoreCount              int                   `json:"core_count,omitempty"`
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
	job := cron.New()
	// CPU 信息上报
	job.AddFunc("0 0/1 * * * ?", func() {
		u4 := uuid.New()
		info := &MonitorInfo{
			Serial:       getCachedProperty("ro.serialno"),
			Uid:          u4.String(),

		}
		log.Infof("CPU information report [%s]\n", addr)
		cpuInfo, err := parseAllTopCPUInfo()
		if err != nil {
			log.Error(err)
		}
		info.Timestamp = time.Now().Unix()
		info.Data = cpuInfo
		info.CoreCount = CPUCoreCount()
		info.Sdk, _ = strconv.Atoi(getCachedProperty("ro.build.version.sdk"))
		str, err := json.Marshal(info)
		if err != nil {
			log.Error(err)
		}
		cpu, err := reportServer(addr+"/monitoring/post_cpu_info", str, "application/json")
		if err != nil {
			log.Error(err)
		}
		log.Infof("CPU information report result [%s]\n", cpu)
	})
	// 内存信息上报
	job.AddFunc("0 0/1 * * * ?", func() {
		u4 := uuid.New()
		info := &MonitorInfo{
			Serial:       getCachedProperty("ro.serialno"),
			Uid:          u4.String(),
		}
		log.Infof("Memory information report [%s]\n", addr)
		memoryInfo, err := parseAllMemoryInfo()
		if err != nil {
			log.Error(err)
		}
		info.Timestamp = time.Now().Unix()
		info.Data = memoryInfo
		str, err := json.Marshal(info)
		if err != nil {
			log.Error(err)
		}
		memory, err := reportServer(addr+"/monitoring/post_mem_info", str, "application/json")
		if err != nil {
			log.Error(err)
		}
		log.Infof("Memory information report result [%s]\n", memory)
	})
	// CPU 温度信息上报
	job.AddFunc("0 0/1 * * * ?", func() {
		u4 := uuid.New()
		info := &MonitorInfo{
			Serial:       getCachedProperty("ro.serialno"),
			Uid:          u4.String(),
		}
		log.Infof("CPU temperature information report [%s]\n", addr)
		tempInfo, err := parseCPUTempInfo()
		if err != nil {
			log.Error(err)
		}
		info.Timestamp = time.Now().Unix()
		info.Data = tempInfo
		str, err := json.Marshal(info)
		if err != nil {
			log.Error(err)
		}
		temp, err := reportServer(addr+"/monitoring/post_temp_info", str, "application/json")
		if err != nil {
			log.Error(err)
		}
		log.Infof("CPU temperature information report result [%s]\n", temp)
	})
	go job.Start()
}

// IP addr 上报
func parseIPInfo(addr string, ip string) (error){
	log.Infof("IP地址上报 [%s]\n", addr)
	deviceInfo := getDeviceInfo()
	reflect.ValueOf(deviceInfo).Elem().Field(8).SetString(ip)
	str, err := json.Marshal(deviceInfo)
	if err != nil {
		return err
	}
	body, err := reportServer(addr + "/phone/phone_report", str, "application/json")
	log.Infof("IP地址上报结果 [%s] \n", body)
	return err
}

// parse all Memory Info
func parseAllMemoryInfo() (info string, err error) {
	output, err := Command{
		Args:    []string{"dumpsys", "meminfo", "--local"},
		Timeout: 60 * time.Second,
	}.CombinedOutputString()
	if err != nil {
		return
	}

	index := strings.Index(output, "Applications Memory Usage")
	if index == -1 {
		err = errors.New("dumpsys meminfo has no [Applications Memory Usage]")
		return
	}
	info = output
	return
}

// parse all top CPU Info
func parseAllTopCPUInfo() (info string, err error){
	output, err := Command{
		Args:    []string{"top", "-b", "-n", "1", "-d", "1"},
		Timeout: 10 * time.Second,
	}.CombinedOutputString()
	if err != nil {
		return
	}

	index := strings.Index(output, "Tasks")
	if index == -1 {
		err = errors.New("top CPU has no [Tasks]")
		return
	}
	info = output
	return
}

// parse CPU temperature Info
func parseCPUTempInfo() (info string, err error){
	output, err := Command{
		Args:    []string{"cat", "/sys/class/thermal/thermal_zone0/temp"},
		Timeout: 10 * time.Second,
	}.CombinedOutputString()
	if err != nil {
		return
	}

	if output == "" {
		err = errors.New("cat CPU temperature error")
		return
	}
	info = output
	return
}