package proxy

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// fakeConn is a scripted sdk_mcp.Connection used to drive replayHandshake.
type fakeConn struct {
	mu        sync.Mutex
	writes    []jsonrpc.Message
	failWrite map[int]error // 0-based write index -> error
	readErr   error
	closed    bool
}

func (c *fakeConn) Read(context.Context) (jsonrpc.Message, error) {
	if c.readErr != nil {
		return nil, c.readErr
	}
	return &jsonrpc.Response{}, nil
}

func (c *fakeConn) Write(_ context.Context, m jsonrpc.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.failWrite[len(c.writes)]; err != nil {
		return err
	}
	c.writes = append(c.writes, m)
	return nil
}

func (c *fakeConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *fakeConn) SessionID() string { return "" }

func mustDecode(t *testing.T, raw string) jsonrpc.Message {
	t.Helper()
	m, err := jsonrpc.DecodeMessage([]byte(raw))
	if err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return m
}

func TestReplayHandshake(t *testing.T) {
	initMsg := mustDecode(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	notif := mustDecode(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	boom := errors.New("boom")

	tests := []struct {
		name       string
		init, note jsonrpc.Message
		conn       *fakeConn
		wantErr    string
		wantWrites int
	}{
		{name: "nothing cached", conn: &fakeConn{}},
		{name: "initialize write fails", init: initMsg, note: notif, conn: &fakeConn{failWrite: map[int]error{0: boom}}, wantErr: "replay initialize failed"},
		{name: "initialize response read fails", init: initMsg, note: notif, conn: &fakeConn{readErr: io.EOF}, wantErr: "read replay initialize response failed", wantWrites: 1},
		{name: "partial replay: notification write fails", init: initMsg, note: notif, conn: &fakeConn{failWrite: map[int]error{1: boom}}, wantErr: "replay initialized notification failed", wantWrites: 1},
		{name: "initialize only", init: initMsg, conn: &fakeConn{}, wantWrites: 1},
		{name: "full replay", init: initMsg, note: notif, conn: &fakeConn{}, wantWrites: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &Runner{cachedInitMsg: tc.init, cachedInitNotif: tc.note}
			err := r.replayHandshake(t.Context(), tc.conn)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
			if len(tc.conn.writes) != tc.wantWrites {
				t.Errorf("writes = %d, want %d", len(tc.conn.writes), tc.wantWrites)
			}
		})
	}
}

func TestInspectClientMessage(t *testing.T) {
	r := &Runner{}
	r.inspectClientMessage(mustDecode(t, `{"jsonrpc":"2.0","id":7,"method":"tools/list"}`))
	if r.cachedInitMsg != nil || r.cachedInitNotif != nil {
		t.Fatal("non-handshake message must not be cached")
	}
	r.inspectClientMessage(mustDecode(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	r.inspectClientMessage(mustDecode(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	if r.cachedInitMsg == nil || r.cachedInitNotif == nil {
		t.Fatalf("handshake not cached: init=%v notif=%v", r.cachedInitMsg, r.cachedInitNotif)
	}
}

func TestSwitchToStandaloneReplayFailureClosesEngine(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(config.Default(dir), dir, strings.NewReader(""), io.Discard, io.Discard)
	r.cachedInitMsg = mustDecode(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)

	// A cancelled context makes the replay against the fresh local engine fail.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := r.switchToStandalone(ctx); err == nil {
		t.Fatal("expected replay failure")
	}
	if r.LocalEngineRunning() || r.activeBackend != nil {
		t.Error("failed switch must not install a backend")
	}
	if starts, _, _, _ := r.Stats(); starts != 0 {
		t.Errorf("localStarts = %d, want 0", starts)
	}
}

func TestSwitchToStandaloneIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(config.Default(dir), dir, strings.NewReader(""), io.Discard, io.Discard)
	if err := r.switchToStandalone(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.localEngine.Close() })
	if err := r.switchToStandalone(t.Context()); err != nil {
		t.Fatal(err)
	}
	if starts, _, _, _ := r.Stats(); starts != 1 {
		t.Errorf("localStarts = %d, want 1", starts)
	}
}

func TestSwitchWaitsForConcurrentSwitch(t *testing.T) {
	dir := t.TempDir()

	t.Run("proxy times out", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		r.switching = true
		if err := r.switchToProxy(t.Context()); err == nil || !strings.Contains(err.Error(), "timeout waiting") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("standalone times out", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		r.switching = true
		if err := r.switchToStandalone(t.Context()); err == nil || !strings.Contains(err.Error(), "timeout waiting") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("proxy already active", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		r.isProxy = true
		if err := r.switchToProxy(t.Context()); err != nil {
			t.Fatal(err)
		}
	})

	completeAfterWait := func(t *testing.T, r *Runner, finish func(), call func() error) {
		t.Helper()
		r.switching = true
		errCh := make(chan error, 1)
		go func() { errCh <- call() }()
		// Let the other "switch" complete; the waiter observes it on its next poll.
		r.mu.Lock()
		finish()
		r.switching = false
		r.mu.Unlock()
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("waiter returned %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("waiter did not return")
		}
	}

	t.Run("proxy completes while waiting", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		completeAfterWait(t, r, func() { r.isProxy = true }, func() error { return r.switchToProxy(t.Context()) })
	})

	t.Run("standalone completes while waiting", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		completeAfterWait(t, r, func() { r.localEngine = &localEngine{} }, func() error { return r.switchToStandalone(t.Context()) })
	})
}

func TestDaemonBaseURL(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{"127.0.0.1", 9000, "http://127.0.0.1:9000"},
		{"localhost", 9000, "http://localhost:9000"},
		{"::1", 9000, "http://[::1]:9000"},
		{"127.0.0.2", 0, "http://127.0.0.2:8080"},
		{"0.0.0.0", 9000, "http://127.0.0.1:9000"},
		{"example.com", -1, "http://127.0.0.1:8080"},
		{"", 9000, "http://127.0.0.1:9000"},
	}
	for _, tc := range tests {
		t.Run(tc.host, func(t *testing.T) {
			cfg := config.Default(".")
			cfg.Server.Host = tc.host
			cfg.Server.Port = tc.port
			r := &Runner{cfg: cfg}
			if got := r.daemonBaseURL(); got != tc.want {
				t.Errorf("daemonBaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSameDir(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		link = ""
	}

	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"empty a", "", dir, false},
		{"empty b", dir, "", false},
		{"identical", dir, dir, true},
		{"unclean path", dir + "/./", dir, true},
		{"different dirs", dir, other, false},
		{"missing dir", filepath.Join(dir, "nope"), other, false},
	}
	if link != "" {
		tests = append(tests, struct {
			name string
			a, b string
			want bool
		}{"symlink to same dir", link, dir, true})
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameDir(tc.a, tc.b); got != tc.want {
				t.Errorf("sameDir(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestLocalEngineCloseNilAndTwice(t *testing.T) {
	var nilEngine *localEngine
	nilEngine.Close() // must not panic

	cancelled := 0
	le := &localEngine{cancel: func() { cancelled++ }}
	le.Close()
	le.Close()
	if cancelled != 1 {
		t.Errorf("cancel called %d times, want 1", cancelled)
	}
}

func TestNewRunnerDefaults(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(nil, dir, nil, nil, nil)
	if r.cfg == nil || r.in != os.Stdin || r.out != os.Stdout || r.stderr != os.Stderr {
		t.Errorf("unexpected defaults: %+v", r)
	}
	if r.IsProxyMode() || r.LocalEngineRunning() {
		t.Error("fresh runner must be idle")
	}
}
