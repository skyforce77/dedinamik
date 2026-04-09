package activity

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

type HTTPAwaitActivity struct {
	From     string `json:"from"`     // listen address for reverse proxy
	To       string `json:"to"`       // target URL to proxy to
	Interval int    `json:"interval"` // health check interval in seconds
}

func (act *HTTPAwaitActivity) GetActivityType() ActivityType {
	return HTTPActivity
}

func StartHTTPListener(ctx context.Context, tracker PeerTracker, act *HTTPAwaitActivity) {
	target, err := url.Parse(act.To)
	if err != nil {
		log.Printf("failed to parse target URL %s: %v", act.To, err)
		return
	}

	var activeRequests atomic.Int64
	proxy := httputil.NewSingleHostReverseProxy(target)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker.PeerConnected()
		activeRequests.Add(1)
		defer func() {
			activeRequests.Add(-1)
			tracker.PeerDisconnected()
		}()
		proxy.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:              act.From,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() { // #nosec G118 -- need independent context for graceful shutdown
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("started HTTP listener %s -> %s", act.From, act.To)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("HTTP listener error on %s: %v", act.From, err)
	}
}
