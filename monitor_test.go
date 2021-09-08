package main

import (
	"testing"
	"time"
)


func TestMonitor(t *testing.T) {

	monitor("http://192.168.50.177:5000")
	time.Sleep(3 * time.Minute)
}
