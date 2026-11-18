package daemon_test

import (
	"context"
	"errors"
	"github.com/sagernet/sing-box/experimental/libbox"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	S "github.com/sagernet/sing-box/adapter/service"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/service"
)

type closeFailureService struct {
	closes    *atomic.Int32
	fail      bool
	startFail bool
	entered   chan struct{}
	release   chan struct{}
}

func (*closeFailureService) Type() string { return "test-close" }
func (*closeFailureService) Tag() string  { return "test" }
func (s *closeFailureService) Start(stage adapter.StartStage, scope *adapter.Scope) error {
	if stage == adapter.StartStateStart {
		if s.entered != nil {
			close(s.entered)
			<-scope.Context().Done()
			<-s.release
		}
		if s.startFail {
			return errors.New("injected startup failure")
		}
	}
	if stage == adapter.StartStateInitialize {
		scope.Add(func() error {
			s.closes.Add(1)
			if s.fail {
				return errors.New("injected native cleanup failure")
			}
			return nil
		})
	}
	return nil
}
func makeCloseTestService(t *testing.T, fail, startFail bool, entered, release chan struct{}) (*daemon.StartedService, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	ctx := include.Context(context.Background())
	registry := S.NewRegistry()
	closes, creates := new(atomic.Int32), new(atomic.Int32)
	S.Register(registry, "test-close", func(_ context.Context, _ log.ContextLogger, _ string, _ struct{}) (adapter.Service, error) {
		creates.Add(1)
		return &closeFailureService{closes: closes, fail: fail, startFail: startFail, entered: entered, release: release}, nil
	})
	ctx = service.ContextWith[adapter.ServiceRegistry](ctx, registry)
	ctx = service.ContextWith[option.ServiceOptionsRegistry](ctx, registry)
	s := daemon.NewStartedService(daemon.ServiceOptions{Context: ctx, LogMaxLines: 10})
	t.Cleanup(func() { _ = s.CloseService(); s.Close() })

	return s, closes, creates
}
func requireCloseFailure(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "injected native cleanup failure") {
		t.Fatalf("cleanup error lost: %v", err)
	}
}
func TestCloseServicePropagatesAndLatchesCleanupFailure(t *testing.T) {
	s, closes, _ := newCloseTestService(t, true)
	requireCloseFailure(t, s.CloseService())
	requireCloseFailure(t, s.CloseService())
	requireCloseFailure(t, s.StartOrReloadService(context.Background(), `{}`, nil))
	if closes.Load() != 1 {
		t.Fatalf("cleanup retried %d times", closes.Load())
	}
}
func TestReloadRejectsReplacementAfterCloseFailure(t *testing.T) {
	s, closes, creates := newCloseTestService(t, true)
	requireCloseFailure(t, s.StartOrReloadService(context.Background(), `{"log":{"disabled":true},"experimental":{"cache_file":{"enabled":false}},"services":[{"type":"test-close","tag":"test"}]}`, nil))
	requireCloseFailure(t, s.CloseService())
	if closes.Load() != 1 || creates.Load() != 1 {
		t.Fatalf("close=%d create=%d", closes.Load(), creates.Load())
	}
}
func TestCloseServiceSuccessIsIdempotent(t *testing.T) {
	s, closes, _ := newCloseTestService(t, false)
	if err := s.CloseService(); err != nil {
		t.Fatal(err)
	}
	if err := s.CloseService(); err != nil {
		t.Fatal(err)
	}
	if closes.Load() != 1 {
		t.Fatalf("close=%d", closes.Load())
	}
}

const closeTestConfig = `{"log":{"disabled":true},"experimental":{"cache_file":{"enabled":false}},"services":[{"type":"test-close","tag":"test"}]}`

func newCloseTestService(t *testing.T, fail bool) (*daemon.StartedService, *atomic.Int32, *atomic.Int32) {
	s, c, n := makeCloseTestService(t, fail, false, nil, nil)
	if err := s.StartOrReloadService(context.Background(), closeTestConfig, nil); err != nil {
		t.Fatal(err)
	}
	return s, c, n
}
func TestFailedStartPreservesCleanupError(t *testing.T) {
	s, c, _ := makeCloseTestService(t, true, true, nil, nil)
	err := s.StartOrReloadService(context.Background(), closeTestConfig, nil)
	requireCloseFailure(t, err)
	if !strings.Contains(err.Error(), "injected startup failure") {
		t.Fatal(err)
	}
	requireCloseFailure(t, s.CloseService())
	if c.Load() != 1 {
		t.Fatalf("close=%d", c.Load())
	}
}
func TestInterruptedStartPreservesCleanupError(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	s, c, _ := makeCloseTestService(t, true, false, entered, release)
	result := make(chan error, 1)
	go func() { result <- s.StartOrReloadService(context.Background(), closeTestConfig, nil) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("start did not enter")
	}
	stopped := make(chan error, 1)
	go func() { stopped <- s.CloseService() }()
	// Ensure CloseService's interruptStart has canceled the production instance context.
	// The test service checks Scope.Context(), so no sleep decides the interleaving.
	close(release)
	requireCloseFailure(t, <-result)
	requireCloseFailure(t, <-stopped)
	if c.Load() != 1 {
		t.Fatalf("close=%d", c.Load())
	}
}
func TestLibboxFacadeReturnsNativeCloseError(t *testing.T) {
	s, _, _ := newCloseTestService(t, true)
	server := &libbox.CommandServer{StartedService: s}
	requireCloseFailure(t, server.CloseService())
}
