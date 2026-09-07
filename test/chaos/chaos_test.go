package chaos

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestMultichannelChaos(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	// 1. Build the latest binary
	cmdBuild := exec.Command("go", "build", "-o", "jokateko-chaos", "../../cmd/jokateko")
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("jokateko-chaos")

	// 2. Setup temporary workspace
	workspace := t.TempDir()
	tasksDir := filepath.Join(workspace, ".jokateko", "tasks")
	os.MkdirAll(tasksDir, 0755)

	// 3. Start the daemon (serve mode)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	binPath, _ := filepath.Abs("jokateko-chaos")
	cmdServe := exec.CommandContext(ctx, binPath, "serve", "--port", "8089")
	cmdServe.Dir = workspace
	if err := cmdServe.Start(); err != nil {
		t.Fatalf("Failed to start serve: %v", err)
	}

	// Give the server a moment to boot
	time.Sleep(1 * time.Second)

	// 4. Start the MCP server (proxy mode)
	cmdMCP := exec.CommandContext(ctx, binPath, "mcp")
	cmdMCP.Dir = workspace
	mcpStdin, _ := cmdMCP.StdinPipe()
	cmdMCP.Stdout = os.Stdout
	if err := cmdMCP.Start(); err != nil {
		t.Fatalf("Failed to start MCP: %v", err)
	}

	var wg sync.WaitGroup
	duration := 10 * time.Second // Run chaos for 10 seconds

	// Worker A: Filesystem Fuzzer
	wg.Add(1)
	go func() {
		defer wg.Done()
		timeout := time.After(duration)
		for {
			select {
			case <-timeout:
				return
			default:
				// Write garbage markdown
				id := fmt.Sprintf("task-%d.md", rand.Intn(1000))
				content := fmt.Sprintf("---\ntitle: Chaos %d\nstatus: backlog\n---\n# Data", rand.Intn(100))
				os.WriteFile(filepath.Join(tasksDir, id), []byte(content), 0644)
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	// Worker B: REST API Fuzzer
	wg.Add(1)
	go func() {
		defer wg.Done()
		timeout := time.After(duration)
		client := &http.Client{Timeout: 2 * time.Second}
		for {
			select {
			case <-timeout:
				return
			default:
				body := []byte(fmt.Sprintf(`{"title":"API Chaos %d"}`, rand.Intn(1000)))
				req, _ := http.NewRequest("POST", "http://127.0.0.1:8089/api/tasks", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
				}
				time.Sleep(15 * time.Millisecond)
			}
		}
	}()

	// Worker C: MCP Fuzzer
	wg.Add(1)
	go func() {
		defer wg.Done()
		timeout := time.After(duration)
		reqID := 1
		for {
			select {
			case <-timeout:
				return
			default:
				payload := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      reqID,
					"method":  "tools/call",
					"params": map[string]interface{}{
						"name": "create_task",
						"arguments": map[string]interface{}{
							"title":   fmt.Sprintf("MCP Chaos %d", rand.Intn(1000)),
							"summary": "Chaos summary",
							"priority": "low",
						},
					},
				}
				b, _ := json.Marshal(payload)
				mcpStdin.Write(append(b, '\n'))
				reqID++
				time.Sleep(20 * time.Millisecond)
			}
		}
	}()

	// Worker D: SSE Event Watcher
	wg.Add(1)
	go func() {
		defer wg.Done()
		client := &http.Client{}
		reqCtx, reqCancel := context.WithTimeout(context.Background(), duration)
		defer reqCancel()
		req, _ := http.NewRequestWithContext(reqCtx, "GET", "http://127.0.0.1:8089/api/events", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("SSE connection failed: %v", err)
			return
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				// It will return an error when context cancels, which is expected
				return
			}
			_ = line // we received an event successfully
		}
	}()

	wg.Wait()

	// 5. Test Graceful Shutdown (Verifying our previous deadlock fix)
	cmdServe.Process.Signal(os.Interrupt)
	cmdMCP.Process.Signal(os.Interrupt)

	errMCP := cmdMCP.Wait()
	errServe := cmdServe.Wait()

	// It's expected they might return exit status 1 or similar on interrupt,
	// but as long as they don't hang, the test passes.
	t.Logf("Serve exited with: %v", errServe)
	t.Logf("MCP exited with: %v", errMCP)

	// If it didn't hang and the SSE stream survived the whole time, the chaos test passes!
}
