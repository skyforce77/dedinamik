package activity

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type ActivityType uint8

const (
	SocketActivity ActivityType = iota
	HTTPActivity
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
		if err := json.Unmarshal(message, s); err != nil {
			log.Printf("failed to parse socket activity: %v", err)
			return nil
		}
		return s
	case "http":
		h := &HTTPAwaitActivity{}
		if err := json.Unmarshal(message, h); err != nil {
			log.Printf("failed to parse http activity: %v", err)
			return nil
		}
		return h
	default:
		return nil
	}
}

// PeerTracker is implemented by MonitoredPlugin to track peer connections.
type PeerTracker interface {
	PeerConnected()
	PeerDisconnected()
}

// PluginWithActivity represents a plugin that has activity listeners.
type PluginWithActivity interface {
	PeerTracker
	GetAwaitList() []AwaitActivity
}

type SocketAwaitActivity struct {
	Connection string `json:"connection"`
	From       string `json:"from"`
	To         string `json:"to"`
}

func (act *SocketAwaitActivity) GetActivityType() ActivityType {
	return SocketActivity
}

func StartSocketListener(ctx context.Context, tracker PeerTracker, act *SocketAwaitActivity) {
	listener, err := net.Listen(act.Connection, act.From)
	if err != nil {
		log.Printf("failed to start socket listener on %s: %v", act.From, err)
		return
	}
	defer listener.Close()

	log.Println("started socket listener", act.From)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("accept error on %s: %v", act.From, err)
				continue
			}
		}

		tracker.PeerConnected()

		toConn, err := dialWithTimeout(act.Connection, act.To, 30*time.Second)
		if err != nil {
			log.Printf("failed to connect to %s after timeout: %v", act.To, err)
			conn.Close()
			tracker.PeerDisconnected()
			continue
		}

		go proxyConnections(tracker, conn, toConn)
	}
}

func dialWithTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	delay := 30 * time.Millisecond
	for {
		conn, err := net.DialTimeout(network, address, 5*time.Second)
		if err == nil {
			return conn, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(delay)
		if delay < 1*time.Second {
			delay *= 2
		}
	}
}

func proxyConnections(tracker PeerTracker, client, target net.Conn) {
	var once sync.Once
	cleanup := func() {
		tracker.PeerDisconnected()
		client.Close()
		target.Close()
	}

	go func() {
		io.Copy(target, client)
		once.Do(cleanup)
	}()

	io.Copy(client, target)
	once.Do(cleanup)
}

func ListenActivity(ctx context.Context, plugins []PluginWithActivity) {
	log.Println("initializing listeners")

	for _, p := range plugins {
		for _, await := range p.GetAwaitList() {
			if act, ok := await.(*SocketAwaitActivity); ok {
				go StartSocketListener(ctx, p, act)
			}
			if act, ok := await.(*HTTPAwaitActivity); ok {
				go StartHTTPListener(ctx, p, act)
			}
		}
	}
}
