package pbft

import (
	"errors"
	"log"
	"time"
)

const IntMin int = ^0x3f3f3f3f

type Client struct {
	Count     int
	StartTime int64
	Msg       chan int64
}

func NewClient() *Client {
	return &Client{Count: IntMin, Msg: make(chan int64)}
}

func (client *Client) Start() error {
	if client.Count >= 0 {
		return errors.New("another request is running")
	}
	client.StartTime = time.Now().UnixMicro()
	log.Println("client started!", client.StartTime, time.Now().UnixMicro())
	//fmt.Println(client.StartTime)
	client.Count = 0
	return nil
}

func (client *Client) End() {
	client.Count = IntMin
	log.Println("client ended!", client.StartTime, time.Now().UnixMicro())
	client.Msg <- time.Now().UnixMicro() - client.StartTime
}
