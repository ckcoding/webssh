package controller

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

func echoSSHServer(t *testing.T, inventoryDelay ...time.Duration) (int, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	auths, execs := &atomic.Int32{}, &atomic.Int32{}
	config := &ssh.ServerConfig{PasswordCallback: func(_ ssh.ConnMetadata, _ []byte) (*ssh.Permissions, error) {
		time.Sleep(20 * time.Millisecond)
		auths.Add(1)
		return nil, nil
	}}
	config.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var connections []net.Conn
	t.Cleanup(func() {
		listener.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range connections {
			conn.Close()
		}
	})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			connections = append(connections, conn)
			mu.Unlock()
			go func() {
				defer conn.Close()
				server, channels, requests, err := ssh.NewServerConn(conn, config)
				if err != nil {
					return
				}
				defer server.Close()
				go ssh.DiscardRequests(requests)
				for incoming := range channels {
					if incoming.ChannelType() != "session" {
						incoming.Reject(ssh.UnknownChannelType, "session only")
						continue
					}
					channel, requests, err := incoming.Accept()
					if err != nil {
						continue
					}
					go func() {
						defer channel.Close()
						for request := range requests {
							switch request.Type {
							case "pty-req", "window-change":
								request.Reply(true, nil)
							case "shell":
								request.Reply(true, nil)
								go func() {
									defer channel.Close()
									channel.Write([]byte("$ "))
									buf := make([]byte, 64*1024)
									for {
										n, err := channel.Read(buf)
										if n > 0 {
											if string(buf[:n]) == "__test_exit__\n" {
												channel.Write([]byte("final output before exit"))
												channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
												return
											}
											if _, writeErr := channel.Write(buf[:n]); writeErr != nil {
												return
											}
										}
										if err != nil {
											return
										}
									}
								}()
							case "exec":
								execs.Add(1)
								var payload struct{ Command string }
								if ssh.Unmarshal(request.Payload, &payload) != nil {
									request.Reply(false, nil)
									continue
								}
								request.Reply(true, nil)
								go func(command string) {
									defer channel.Close()
									if strings.Contains(command, "uname -s") {
										channel.Write([]byte("Linux\n"))
									} else {
										io.ReadAll(channel)
										if len(inventoryDelay) > 0 {
											time.Sleep(inventoryDelay[0])
										}
										channel.Write([]byte("hostname=fixture\nos=Fixture Linux\ncpu_cores=4\nmem_total_kb=8388608\nmem_available_kb=4194304\n"))
									}
									channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
								}(payload.Command)
							default:
								request.Reply(false, nil)
							}
						}
					}()
				}
			}()
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port, auths, execs
}

func terminalHTTPServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/ssh/check", func(c *gin.Context) { c.JSON(http.StatusOK, CheckSSHConnection(c)) })
	router.POST("/ssh/session", func(c *gin.Context) { c.JSON(http.StatusOK, CreateSSHSession(c)) })
	router.GET("/term", func(c *gin.Context) { TermWs(c, time.Minute) })
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

func postConnection(t *testing.T, endpoint string, payload interface{}) map[string]interface{} {
	t.Helper()
	content, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(endpoint, "application/json", bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result struct {
		Msg  string
		Data map[string]interface{}
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Msg != "success" {
		t.Fatalf("connection failed: %s", result.Msg)
	}
	return result.Data
}

func TestTerminalPreparedHandshakeAndBidirectionalFlow(t *testing.T) {
	for _, prepared := range []bool{false, true} {
		t.Run(fmtPrepared(prepared), func(t *testing.T) {
			port, auths, execs := echoSSHServer(t)
			server := terminalHTTPServer(t)
			started := time.Now()
			payload := map[string]interface{}{"host": "127.0.0.1", "username": "test", "password": "test", "port": port, "probeOnly": true, "prepareSession": prepared}
			data := postConnection(t, server.URL+"/ssh/check", payload)
			if !prepared {
				data = postConnection(t, server.URL+"/ssh/session", payload)
			}
			sessionID, ok := data["sessionId"].(string)
			if !ok {
				t.Fatal("missing connection session")
			}
			params := url.Values{"sessionId": {sessionID}, "rows": {"24"}, "cols": {"80"}, "flowControl": {"1"}}
			dialer := websocket.Dialer{EnableCompression: true}
			ws, _, err := dialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/term?"+params.Encode(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer ws.Close()
			ws.SetReadDeadline(time.Now().Add(10 * time.Second))
			_, prompt, err := ws.ReadMessage()
			if err != nil || string(prompt) != "$ " {
				t.Fatalf("prompt=%q error=%v", prompt, err)
			}
			wantAuths := int32(2)
			if prepared {
				wantAuths = 1
			}
			if auths.Load() != wantAuths || execs.Load() != 0 {
				t.Fatalf("handshakes=%d metadata commands=%d", auths.Load(), execs.Load())
			}
			t.Logf("handshakes=%d time to prompt=%s", auths.Load(), time.Since(started).Round(time.Millisecond))
			if !prepared {
				return
			}
			// An echoing shell stresses both directions: blocking the SSH stdin
			// writer must not stop output acknowledgements or deadlock a large paste.
			input := []byte("resize:24:80\r" + strings.Repeat("中文🙂", 800000))
			offset, pendingInput := 0, 0
			sendAvailable := func() {
				for offset < len(input) && pendingInput < 256*1024 {
					n := min(64*1024, len(input)-offset, 256*1024-pendingInput)
					if err := ws.WriteMessage(websocket.BinaryMessage, input[offset:offset+n]); err != nil {
						t.Fatal(err)
					}
					offset += n
					pendingInput += n
				}
			}
			sendAvailable()
			output := make([]byte, 0, len(input))
			pendingAck := 0
			started = time.Now()
			for len(output) < len(input) {
				kind, data, err := ws.ReadMessage()
				if err != nil {
					t.Fatalf("paste stalled after %d/%d bytes: %v", len(output), len(input), err)
				}
				if kind == websocket.TextMessage && bytes.HasPrefix(data, []byte("\x00inputack:")) {
					n, err := strconv.Atoi(string(data[10:]))
					if err != nil || n <= 0 || n > pendingInput {
						t.Fatal("invalid input acknowledgement")
					}
					pendingInput -= n
					sendAvailable()
					continue
				}
				output = append(output, data...)
				pendingAck += len(data)
				if pendingAck >= 64*1024 {
					if err := ws.WriteMessage(websocket.TextMessage, []byte("\x00ack:"+strconv.Itoa(pendingAck))); err != nil {
						t.Fatal(err)
					}
					pendingAck = 0
				}
			}
			if !bytes.Equal(output, input) {
				t.Fatal("large paste or UTF-8 output was corrupted")
			}
			t.Logf("%d bytes echoed intact in %s", len(input), time.Since(started).Round(time.Millisecond))
		})
	}
}

func fmtPrepared(prepared bool) string {
	if prepared {
		return "reused_handshake"
	}
	return "legacy_handshake"
}

func TestTerminalFlushesOutputBeforeShellExit(t *testing.T) {
	for _, flowControl := range []string{"", "1"} {
		t.Run("flow_"+flowControl, func(t *testing.T) {
			port, _, _ := echoSSHServer(t)
			server := terminalHTTPServer(t)
			payload := map[string]interface{}{"host": "127.0.0.1", "username": "test", "password": "test", "port": port, "prepareSession": true}
			data := postConnection(t, server.URL+"/ssh/check", payload)
			params := url.Values{"sessionId": {data["sessionId"].(string)}, "rows": {"24"}, "cols": {"80"}, "flowControl": {flowControl}}
			ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/term?"+params.Encode(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer ws.Close()
			ws.SetReadDeadline(time.Now().Add(3 * time.Second))
			if _, _, err := ws.ReadMessage(); err != nil {
				t.Fatal(err)
			}
			kind := websocket.TextMessage
			if flowControl == "1" {
				kind = websocket.BinaryMessage
			}
			if err := ws.WriteMessage(kind, []byte("__test_exit__\n")); err != nil {
				t.Fatal(err)
			}
			var output []byte
			for {
				kind, data, err := ws.ReadMessage()
				if err != nil {
					break
				}
				if kind == websocket.BinaryMessage {
					output = append(output, data...)
				}
			}
			if string(output) != "final output before exit" {
				t.Fatalf("final output lost: %q", output)
			}
		})
	}
}
