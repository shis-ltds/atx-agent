package main

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

type MockConn struct {
	buffer *bytes.Buffer
}

func (c *MockConn) Read(b []byte) (n int, err error) {
	return c.buffer.Read(b)
}

func (c *MockConn) Write(b []byte) (n int, err error) {
	return c.buffer.Write(b)
}

func (c *MockConn) Close() error                       { return nil }
func (c *MockConn) LocalAddr() net.Addr                { return nil }
func (c *MockConn) RemoteAddr() net.Addr               { return nil }
func (c *MockConn) SetDeadline(t time.Time) error      { return nil }
func (c *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *MockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestDrainTouchRequests(t *testing.T) {
	reqC := make(chan TouchRequest, 0)
	conn := &MockConn{
		buffer: bytes.NewBuffer(nil),
	}
	err := drainTouchRequests(conn, reqC)
	assert.Error(t, err)

	conn = &MockConn{
		buffer: bytes.NewBufferString(`v 1
^ 10 1080 1920 255
$ 25654`),
	}
	reqC = make(chan TouchRequest, 4)
	reqC <- TouchRequest{
		Operation: "d",
		Index:     1,
		PercentX:  1.0,
		PercentY:  1.0,
		Pressure:  1,
	}
	reqC <- TouchRequest{
		Operation: "c",
	}
	reqC <- TouchRequest{
		Operation: "m",
		Index:     3,
		PercentX:  0.5,
		PercentY:  0.5,
		Pressure:  1,
	}
	reqC <- TouchRequest{
		Operation: "u",
		Index:     4,
	}
	close(reqC)
	drainTouchRequests(conn, reqC)
	output := string(conn.buffer.Bytes())
	assert.Equal(t, "d 1 1080 1920 255\nc\nm 3 540 960 255\nu 4\n", output)
}

func TestJsonRes(t *testing.T) {
	data := "{\"code\":2000,\"data\":{\"url\":\"http://192.168.1.154:5000/report/perf\"},\"message\":\"Success\"}"

	if err := json.Unmarshal([]byte(data), &res); err != nil {
		log.Infof("IP地址上报失败 [%s] err [%s] \n", data, err.Error())
	}
	assert.Equal(t, res.Code, 2000)
}
