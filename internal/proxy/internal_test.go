package proxy

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// fakeConn is a scripted sdk_mcp.Connection used to drive replayHandshake.
// Read first drains prelude, then answers the last written request.
type fakeConn struct {
	mu        sync.Mutex
	writes    []jsonrpc.Message
	failWrite map[int]error // 0-based write index -> error
	readErr   error
	prelude   []jsonrpc.Message // returned by Read before the reply
	replyErr  error             // error carried by the reply
	noReply   bool              // block until ctx is done instead of replying
	closed    bool
}

func (c *fakeConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.readErr != nil {
		return nil, c.readErr
	}
	if len(c.prelude) > 0 {
		m := c.prelude[0]
		c.prelude = c.prelude[1:]
		return m, nil
	}
	if c.noReply {
		c.mu.Unlock()
		<-ctx.Done()
		c.mu.Lock()
		return nil, ctx.Err()
	}
	for _, w := range slices.Backward(c.writes) {
		if req, ok := w.(*jsonrpc.Request); ok && req.IsCall() {
			return &jsonrpc.Response{ID: req.ID, Result: []byte(`{}`), Error: c.replyErr}, nil
		}
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

func mustDecodeRequest(t *testing.T, raw string) *jsonrpc.Request {
	t.Helper()
	req, ok := mustDecode(t, raw).(*jsonrpc.Request)
	if !ok {
		t.Fatalf("%s is not a request", raw)
	}
	return req
}

func TestReplayHandshake(t *testing.T) {
	initMsg := mustDecodeRequest(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	notif := mustDecodeRequest(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	boom := errors.New("boom")
	clientID, _ := jsonrpc.MakeID(float64(1))

	tests := []struct {
		name       string
		init, note *jsonrpc.Request
		conn       *fakeConn
		wantErr    string
		wantWrites int
	}{
		{name: "nothing cached", conn: &fakeConn{}},
		{name: "initialize write fails", init: initMsg, note: notif, conn: &fakeConn{failWrite: map[int]error{0: boom}}, wantErr: "replay initialize failed"},
		{name: "initialize response read fails", init: initMsg, note: notif, conn: &fakeConn{readErr: io.EOF}, wantErr: "read replay initialize response failed", wantWrites: 1},
		{name: "initialize rejected", init: initMsg, note: notif, conn: &fakeConn{replyErr: boom}, wantErr: "replay initialize rejected", wantWrites: 1},
		{name: "no response times out", init: initMsg, note: notif, conn: &fakeConn{noReply: true}, wantErr: "read replay initialize response failed", wantWrites: 1},
		{name: "partial replay: notification write fails", init: initMsg, note: notif, conn: &fakeConn{failWrite: map[int]error{1: boom}}, wantErr: "replay initialized notification failed", wantWrites: 1},
		{name: "initialize only", init: initMsg, conn: &fakeConn{}, wantWrites: 1},
		{name: "full replay", init: initMsg, note: notif, conn: &fakeConn{}, wantWrites: 2},
		{
			name: "skips unrelated messages before the response",
			init: initMsg, note: notif,
			conn: &fakeConn{prelude: []jsonrpc.Message{
				mustDecode(t, `{"jsonrpc":"2.0","method":"notifications/message","params":{"level":"info","data":"x"}}`),
				&jsonrpc.Response{ID: clientID, Result: []byte(`{}`)},
			}},
			wantWrites: 2,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &Runner{cachedInitMsg: tc.init, cachedInitNotif: tc.note}
			ctx := t.Context()
			if tc.conn.noReply {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			}
			err := r.replayHandshake(ctx, tc.conn)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
			if len(tc.conn.writes) != tc.wantWrites {
				t.Fatalf("writes = %d, want %d", len(tc.conn.writes), tc.wantWrites)
			}
			if tc.wantWrites == 0 {
				return
			}
			// The replayed initialize uses a proxy-private ID, never the client's.
			sent := tc.conn.writes[0].(*jsonrpc.Request)
			if id, _ := sent.ID.Raw().(string); !strings.HasPrefix(id, replayIDPrefix) {
				t.Errorf("replay ID = %v, want %q prefix", sent.ID.Raw(), replayIDPrefix)
			}
			if sent.Method != "initialize" || string(sent.Params) != string(initMsg.Params) {
				t.Errorf("replayed %s %s, want the cached initialize", sent.Method, sent.Params)
			}
			if initMsg.ID != clientID {
				t.Error("replay must not mutate the cached client request")
			}
			if tc.wantWrites == 2 && tc.conn.writes[1] != notif {
				t.Error("second write must be the cached initialized notification")
			}
		})
	}
}

func TestReplayHandshakeIDsAreUnique(t *testing.T) {
	r := &Runner{cachedInitMsg: mustDecodeRequest(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)}
	conn := &fakeConn{}
	for range 2 {
		if err := r.replayHandshake(t.Context(), conn); err != nil {
			t.Fatal(err)
		}
	}
	a, b := conn.writes[0].(*jsonrpc.Request).ID, conn.writes[1].(*jsonrpc.Request).ID
	if a == b {
		t.Errorf("replays reused ID %v", a.Raw())
	}
}

func TestInspectClientMessage(t *testing.T) {
	r := &Runner{}
	for _, raw := range []string{
		`{"jsonrpc":"2.0","id":7,"method":"tools/list"}`,
		// SEP-2575 clients need no replay; see replayHandshake.
		`{"jsonrpc":"2.0","id":"server-discover-probe-1","method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`,
		`{"jsonrpc":"2.0","id":1,"result":{}}`,
		// An initialize notification is malformed and must not be replayed.
		`{"jsonrpc":"2.0","method":"initialize","params":{}}`,
	} {
		r.inspectClientMessage(mustDecode(t, raw))
	}
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
	r.cachedInitMsg = mustDecodeRequest(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)

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

// TestSwitchWaitsForInFlightSwitch covers a switch requested while another is
// still running, e.g. the proxy write failover racing the tail of a
// switchToProxy that is still closing the old local engine.
func TestSwitchWaitsForInFlightSwitch(t *testing.T) {
	dir := t.TempDir()

	// inFlight holds switchMu like a running switch, runs call, and checks
	// that call blocks until finish completes the other switch.
	inFlight := func(t *testing.T, r *Runner, finish func(), call func() error) {
		t.Helper()
		r.switchMu.Lock()
		errCh := make(chan error, 1)
		go func() { errCh <- call() }()
		select {
		case err := <-errCh:
			r.switchMu.Unlock()
			t.Fatalf("switch did not wait for the in-flight switch: %v", err)
		case <-time.After(50 * time.Millisecond):
		}
		r.mu.Lock()
		finish()
		r.mu.Unlock()
		r.switchMu.Unlock()
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("switch returned %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("switch did not return")
		}
	}

	t.Run("standalone after in-flight proxy switch", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		daemon := &fakeConn{}
		inFlight(t, r, func() {
			r.activeBackend = daemon
			r.isProxy = true
		}, func() error { return r.switchToStandalone(t.Context()) })
		t.Cleanup(func() { r.localEngine.Close() })
		if !r.LocalEngineRunning() {
			t.Fatal("standalone engine not running after the switch")
		}
		if !daemon.closed {
			t.Error("daemon connection not closed")
		}
	})

	t.Run("proxy after in-flight proxy switch", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		inFlight(t, r, func() { r.isProxy = true }, func() error { return r.switchToProxy(t.Context()) })
		if _, _, connects, _ := r.Stats(); connects != 0 {
			t.Errorf("proxyConnects = %d, want 0", connects)
		}
	})

	t.Run("proxy already active", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		r.isProxy = true
		if err := r.switchToProxy(t.Context()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestForward(t *testing.T) {
	dir := t.TempDir()
	call := mustDecode(t, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{}}`)
	notif := mustDecode(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)

	t.Run("delivers to the active backend", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		backend, client := &fakeConn{}, &fakeConn{}
		r.activeBackend = backend
		r.forward(t.Context(), client, call)
		if len(backend.writes) != 1 || len(client.writes) != 0 {
			t.Fatalf("backend writes = %d, client writes = %d", len(backend.writes), len(client.writes))
		}
	})

	t.Run("proxy write failure fails over", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		daemon, client := &fakeConn{failWrite: map[int]error{0: errors.New("connection refused")}}, &fakeConn{}
		r.activeBackend = daemon
		r.isProxy = true
		r.forward(t.Context(), client, call)
		t.Cleanup(func() { r.localEngine.Close() })
		if !r.LocalEngineRunning() {
			t.Fatal("did not fail over to the standalone engine")
		}
		if len(client.writes) != 0 {
			t.Fatalf("client got %v, want the call delivered to the new backend", client.writes)
		}
	})

	t.Run("undeliverable call gets an error response", func(t *testing.T) {
		r := NewRunner(config.Default(dir), dir, nil, nil, io.Discard)
		backend, client := &fakeConn{failWrite: map[int]error{0: errors.New("broken pipe")}}, &fakeConn{}
		r.activeBackend = backend
		r.localEngine = &localEngine{}
		r.forward(t.Context(), client, call)
		r.forward(t.Context(), client, notif)
		if len(client.writes) != 1 {
			t.Fatalf("client writes = %v, want one error response", client.writes)
		}
		resp, ok := client.writes[0].(*jsonrpc.Response)
		if !ok || resp.ID != call.(*jsonrpc.Request).ID {
			t.Fatalf("client got %#v, want response to id 7", client.writes[0])
		}
		if werr, ok := errors.AsType[*jsonrpc.Error](resp.Error); !ok || werr.Code != jsonrpc.CodeInternalError {
			t.Fatalf("response error = %v, want internal error", resp.Error)
		}
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
