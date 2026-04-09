package monitor

import (
	"context"
	"log"
	"time"
)

// Monitorable represents a plugin that can be monitored for idle timeout.
type Monitorable interface {
	GetStateName() string // returns "running", "sleeping", "stopped"
	IsRunning() bool
	WantToSleep() bool
	CanSleep() bool
	Sleep()
	Stop()
}

func Start(ctx context.Context, plugins []Monitorable) {
	log.Println("initializing monitoring")

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, p := range plugins {
					if p.IsRunning() && p.WantToSleep() {
						if p.CanSleep() {
							p.Sleep()
						} else {
							p.Stop()
						}
					}
				}
			}
		}
	}()
}
