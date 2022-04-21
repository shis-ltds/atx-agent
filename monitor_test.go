package main

import (
	"fmt"
	"testing"
	"time"
)

func TestMonitor(t *testing.T) {

	monitor("http://192.168.50.177:5000")
	time.Sleep(3 * time.Minute)
}

func TestMonitorCpuCmd(t *testing.T) {
	cmd := []string{"top", "-b", "-n", "1", "-d", "1"}
	sdk := 28
	if sdk < 28 {
		cmd = append(cmd[:1], cmd[2:]...)
	}
	fmt.Printf("%v", cmd)
}
