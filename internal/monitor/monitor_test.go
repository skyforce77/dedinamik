package monitor

import (
	"context"
	"sync"
	"testing"
	"time"
)

type mockMonitorable struct {
	running    bool
	wantSleep  bool
	canSleep   bool
	sleepCount int
	stopCount  int
	mu         sync.Mutex
}

func (m *mockMonitorable) GetStateName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return "running"
	}
	return "stopped"
}

func (m *mockMonitorable) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *mockMonitorable) WantToSleep() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.wantSleep
}

func (m *mockMonitorable) CanSleep() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.canSleep
}

func (m *mockMonitorable) Sleep() {
	m.mu.Lock()
	m.sleepCount++
	m.mu.Unlock()
}

func (m *mockMonitorable) Stop() {
	m.mu.Lock()
	m.stopCount++
	m.mu.Unlock()
}

func (m *mockMonitorable) getSleepCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sleepCount
}

func (m *mockMonitorable) getStopCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopCount
}

func TestStart_StopsIdlePlugin(t *testing.T) {
	mock := &mockMonitorable{
		running:   true,
		wantSleep: true,
		canSleep:  false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	Start(ctx, []Monitorable{mock})
	time.Sleep(2 * time.Second)

	if got := mock.getStopCount(); got == 0 {
		t.Errorf("expected Stop to be called at least once, got %d calls", got)
	}
	if got := mock.getSleepCount(); got != 0 {
		t.Errorf("expected Sleep not to be called, got %d calls", got)
	}
}

func TestStart_SleepsIdlePlugin(t *testing.T) {
	mock := &mockMonitorable{
		running:   true,
		wantSleep: true,
		canSleep:  true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	Start(ctx, []Monitorable{mock})
	time.Sleep(2 * time.Second)

	if got := mock.getSleepCount(); got == 0 {
		t.Errorf("expected Sleep to be called at least once, got %d calls", got)
	}
	if got := mock.getStopCount(); got != 0 {
		t.Errorf("expected Stop not to be called, got %d calls", got)
	}
}

func TestStart_IgnoresRunningNotIdle(t *testing.T) {
	mock := &mockMonitorable{
		running:   true,
		wantSleep: false,
		canSleep:  false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	Start(ctx, []Monitorable{mock})
	time.Sleep(2 * time.Second)

	if got := mock.getStopCount(); got != 0 {
		t.Errorf("expected Stop not to be called, got %d calls", got)
	}
	if got := mock.getSleepCount(); got != 0 {
		t.Errorf("expected Sleep not to be called, got %d calls", got)
	}
}

func TestStart_IgnoresNonRunning(t *testing.T) {
	mock := &mockMonitorable{
		running:   false,
		wantSleep: true,
		canSleep:  true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	Start(ctx, []Monitorable{mock})
	time.Sleep(2 * time.Second)

	if got := mock.getStopCount(); got != 0 {
		t.Errorf("expected Stop not to be called, got %d calls", got)
	}
	if got := mock.getSleepCount(); got != 0 {
		t.Errorf("expected Sleep not to be called, got %d calls", got)
	}
}

func TestStart_ContextCancel(t *testing.T) {
	mock := &mockMonitorable{
		running:   true,
		wantSleep: true,
		canSleep:  false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	Start(ctx, []Monitorable{mock})

	// Let the monitor tick a couple of times.
	time.Sleep(2 * time.Second)
	cancel()

	// Record the count right after cancellation.
	time.Sleep(200 * time.Millisecond)
	countAfterCancel := mock.getStopCount()

	// Wait and verify no further calls are made.
	time.Sleep(2 * time.Second)
	countLater := mock.getStopCount()

	if countAfterCancel == 0 {
		t.Errorf("expected Stop to be called before cancel, got 0")
	}
	if countLater != countAfterCancel {
		t.Errorf("expected no more Stop calls after cancel, got %d before and %d after", countAfterCancel, countLater)
	}
}
