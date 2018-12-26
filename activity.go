package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"time"
)

type ActivityType uint8

const (
	SocketActivity ActivityType = iota
)

type AwaitActivity interface {
	GetActivityType() ActivityType
}

type AwaitActivityFile struct {
	Type string `json:"type"`
}

func (file *AwaitActivityFile) Create(message json.RawMessage) AwaitActivity {
	switch file.Type {
	case "socket":
		s := &SocketAwaitActivity{}
		json.Unmarshal(message, s)
		return s
	default:
		return nil
	}
}

type SocketAwaitActivity struct {
	Connection string `json:"connection"`
	From       string `json:"from"`
	To         string `json:"to"`
}

func (act *SocketAwaitActivity) GetActivityType() ActivityType {
	return SocketActivity
}

func startSocketListener(plugin *MonitoredPlugin, act *SocketAwaitActivity) {
	listener, err := net.Listen(act.Connection, act.From)
	if err != nil {
		panic(err)
	} else {
		defer listener.Close()
	}

	log.Println("started socket listener", act.From)

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		plugin.PeerConnected()

		err = errors.New("")
		var toConn net.Conn
		for err != nil {
			toConn, err = net.Dial(act.Connection, act.To)
			if err != nil {
				time.Sleep(30 * time.Millisecond)
			}
		}

		go func(plugin *MonitoredPlugin, c net.Conn, d net.Conn) {
			io.Copy(c, d)
			err := c.Close()
			if err != nil {
				plugin.PeerDisconnected()
			}
			toConn.Close()
		}(plugin, conn, toConn)

		go func(plugin *MonitoredPlugin, c net.Conn, d net.Conn) {
			io.Copy(d, c)
			err := c.Close()
			if err != nil {
				plugin.PeerDisconnected()
			}
			toConn.Close()
		}(plugin, conn, toConn)
	}
}

func listenActivity(plugins []*MonitoredPlugin) {
	log.Println("initializing listeners")

	for _, plugin := range plugins {
		for _, await := range plugin.AwaitList {
			if await.GetActivityType() == SocketActivity {
				go startSocketListener(plugin, await.(*SocketAwaitActivity))
			}
		}
	}
}
