package core

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	websocketReadTimeout     = 60 * time.Second
	websocketPingInterval    = 15 * time.Second
	websocketWriteTimeout    = 10 * time.Second
	sshKeepaliveInterval     = 20 * time.Second
	tcpKeepaliveInterval     = 15 * time.Second
	defaultSSHConnectTimeout = 10 * time.Second
)

func normalizeSSHClientAddress(address string) string {
	address = strings.TrimSpace(address)
	if strings.Contains(address, ":") && !strings.HasPrefix(address, "[") {
		address = "[" + address + "]"
	}
	return address
}

// BuildSSHClient 根据输入构建并校验一个 SSH 客户端配置。
func BuildSSHClient(address, username, password string, port, loginType int) (SSHClient, error) {
	client := NewSSHClient()
	client.IPAddress = normalizeSSHClientAddress(address)
	if client.IPAddress == "" {
		return client, fmt.Errorf("host is required")
	}
	client.Username = strings.TrimSpace(username)
	if client.Username == "" {
		return client, fmt.Errorf("username is required")
	}
	if strings.TrimSpace(password) == "" {
		return client, fmt.Errorf("password is required")
	}
	client.Password = password
	if port > 0 {
		if port > 65535 {
			return client, fmt.Errorf("invalid port")
		}
		client.Port = port
	} else if port < 0 {
		return client, fmt.Errorf("invalid port")
	}
	if loginType == 1 {
		client.LoginType = 1
	}
	return client, nil
}

// DecodedMsgToSSHClient 字符串信息解析为ssh客户端
func DecodedMsgToSSHClient(sshInfo string) (SSHClient, error) {
	client := NewSSHClient()
	decoded, err := base64.StdEncoding.DecodeString(sshInfo)
	if err != nil {
		return client, err
	}
	err = json.Unmarshal(decoded, &client)
	if err != nil {
		return client, err
	}
	client.IPAddress = normalizeSSHClientAddress(client.IPAddress)
	if client.Port == 0 {
		client.Port = 22
	} else if client.Port < 0 || client.Port > 65535 {
		return client, fmt.Errorf("invalid port")
	}
	return client, nil
}

func buildHostKeyCallback() (ssh.HostKeyCallback, error) {
	if !strictHostKeyCheckingEnabled() {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	knownHostsPath := strings.TrimSpace(os.Getenv("WEBSSH_KNOWN_HOSTS"))
	if knownHostsPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		knownHostsPath = filepath.Join(homeDir, ".ssh", "known_hosts")
	}
	return knownhosts.New(knownHostsPath)
}

func strictHostKeyCheckingEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("WEBSSH_STRICT_HOST_KEY")))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

// GenerateClient 创建ssh客户端
func (sclient *SSHClient) GenerateClient() error {
	return sclient.generateClient(defaultSSHConnectTimeout)
}

func (sclient *SSHClient) generateClient(connectTimeout time.Duration) error {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		config       ssh.Config
		err          error
	)
	hostKeyCallback, err := buildHostKeyCallback()
	if err != nil {
		return err
	}
	auth = make([]ssh.AuthMethod, 0)
	if sclient.LoginType == 0 {
		auth = append(auth, ssh.Password(sclient.Password))
		// Windows OpenSSH 的密码认证常以 keyboard-interactive 形式提供，
		// 补充该回退方式以提升兼容性。
		pass := sclient.Password
		auth = append(auth, ssh.KeyboardInteractive(func(name, instruction string, questions []string, echos []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range questions {
				answers[i] = pass
			}
			return answers, nil
		}))
	} else {
		if signer, err := ssh.ParsePrivateKey([]byte(sclient.Password)); err != nil {
			return err
		} else {
			auth = append(auth, ssh.PublicKeys(signer))
		}
	}
	config = ssh.Config{
		Ciphers: []string{
			"aes128-gcm@openssh.com",
			"aes256-gcm@openssh.com",
			"chacha20-poly1305@openssh.com",
			"aes128-ctr", "aes192-ctr", "aes256-ctr",
			"arcfour256", "arcfour128", "arcfour",
			"aes128-cbc", "3des-cbc",
		},
		KeyExchanges: []string{
			"curve25519-sha256",
			"curve25519-sha256@libssh.org",
			"ecdh-sha2-nistp256",
			"ecdh-sha2-nistp384",
			"ecdh-sha2-nistp521",
			"diffie-hellman-group14-sha256",
			"diffie-hellman-group16-sha512",
			"diffie-hellman-group-exchange-sha256",
			"diffie-hellman-group14-sha1",
			"diffie-hellman-group1-sha1",
		},
		MACs: []string{
			"hmac-sha2-256-etm@openssh.com",
			"hmac-sha2-512-etm@openssh.com",
			"hmac-sha2-256",
			"hmac-sha2-512",
			"hmac-sha1",
			"hmac-sha1-96",
		},
	}
	clientConfig = &ssh.ClientConfig{
		User:            sclient.Username,
		Auth:            auth,
		Timeout:         connectTimeout,
		Config:          config,
		HostKeyCallback: hostKeyCallback,
	}
	addr = fmt.Sprintf("%s:%d", sclient.IPAddress, sclient.Port)
	dialer := &net.Dialer{
		Timeout:   connectTimeout,
		KeepAlive: tcpKeepaliveInterval,
	}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return err
	}
	// ClientConfig.Timeout only covers ssh.Dial; NewClientConn needs a deadline
	// for the greeting, key exchange and authentication as well.
	if err := conn.SetDeadline(time.Now().Add(connectTimeout)); err != nil {
		conn.Close()
		return err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, clientConfig)
	if err != nil {
		conn.Close()
		return err
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		sshConn.Close()
		return err
	}
	client = ssh.NewClient(sshConn, chans, reqs)
	sclient.Client = client
	return nil
}

// InitTerminal 初始化终端
func (sclient *SSHClient) InitTerminal(ws *websocket.Conn, rows, cols int, flowControl ...bool) error {
	sshSession, err := sclient.Client.NewSession()
	if err != nil {
		return err
	}
	sclient.Session = sshSession
	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		sclient.Close()
		return err
	}
	sclient.StdinPipe = stdinPipe
	wsOutput := &wsOutput{
		ws:  ws,
		buf: make([]byte, 0, wsOutputInitialBuf),
	}
	if len(flowControl) > 0 && flowControl[0] {
		wsOutput.flow = newTerminalFlow()
	}
	// 终端标准输出和标准错误都写入同一个 websocket 输出器。
	sshSession.Stdout = wsOutput
	sshSession.Stderr = wsOutput
	sclient.wsWriteMu = &sync.Mutex{}
	wsOutput.mu = sclient.wsWriteMu
	sclient.wsOutput = wsOutput
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}

	if err := sshSession.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		sclient.Close()
		return err
	}
	if err := sshSession.Shell(); err != nil {
		sclient.Close()
		return err
	}
	return nil
}

// RunCommand 执行一次性远端命令
func (sclient *SSHClient) RunCommand(command string) (string, error) {
	if sclient.Client == nil {
		return "", fmt.Errorf("ssh client not connected")
	}
	session, err := sclient.Client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	stdin, err := session.StdinPipe()
	if err != nil {
		return "", err
	}

	if err := session.Start("sh"); err != nil {
		// 回退到标准 Run，如果服务器不支持 Start("sh")
		if err := session.Run(command); err != nil {
			message := strings.TrimSpace(stderr.String())
			if message != "" {
				return "", fmt.Errorf("%w: %s", err, message)
			}
			return "", err
		}
		return stdout.String(), nil
	}

	// 将大脚本通过 stdin 传入，避免部分服务器（如 OpenWrt Dropbear）有 exec 长度限制导致被截断或直接 EOF 断开
	_, err = stdin.Write([]byte(command + "\n"))
	stdin.Close()
	if err != nil {
		return "", err
	}

	if err := session.Wait(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return "", fmt.Errorf("%w: %s", err, message)
		}
		return "", err
	}

	return stdout.String(), nil
}

// RunScript 通过启动指定的 shell（如 "sh" 或 "powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command -"），
// 并将 script 经由 stdin 传入执行。可绕过命令行长度限制（Windows cmd.exe 约 8191 字符），
// 适用于较长的脚本。返回 stdout 内容。
func (sclient *SSHClient) RunScript(shell, script string) (string, error) {
	if sclient.Client == nil {
		return "", fmt.Errorf("ssh client not connected")
	}
	session, err := sclient.Client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	stdin, err := session.StdinPipe()
	if err != nil {
		return "", err
	}

	if err := session.Start(shell); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return "", fmt.Errorf("start %q: %w: %s", shell, err, message)
		}
		return "", fmt.Errorf("start %q: %w", shell, err)
	}

	if _, err := stdin.Write([]byte(script + "\n")); err != nil {
		stdin.Close()
		return "", err
	}
	stdin.Close()

	if err := session.Wait(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return "", fmt.Errorf("%w: %s", err, message)
		}
		return "", err
	}

	return stdout.String(), nil
}

// Connect ws连接
func (sclient *SSHClient) Connect(ws *websocket.Conn, timeout time.Duration, closeTip string) {
	if sclient.Client == nil || sclient.Session == nil || sclient.StdinPipe == nil {
		ws.WriteMessage(websocket.TextMessage, []byte("ssh terminal is not initialized"))
		ws.Close()
		return
	}

	stopCh := make(chan struct{})
	activityCh := make(chan struct{}, 1)
	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			close(stopCh)
		})
	}
	notifyActivity := func() {
		select {
		case activityCh <- struct{}{}:
		default:
		}
	}

	ws.SetReadDeadline(time.Now().Add(websocketReadTimeout))
	ws.SetPongHandler(func(string) error {
		return ws.SetReadDeadline(time.Now().Add(websocketReadTimeout))
	})
	output := sclient.wsOutput
	if output != nil {
		output.setActivityFn(notifyActivity)
	}
	session, client, stdin := sclient.Session, sclient.Client, sclient.StdinPipe
	var inputQueue chan []byte
	if output != nil && output.flow != nil {
		// Four 64 KiB frames fit the browser's input window. A blocked SSH
		// stdin writer must leave the reader free to process output acknowledgements.
		inputQueue = make(chan []byte, 4)
		go func() {
			for {
				select {
				case <-stopCh:
					return
				case p := <-inputQueue:
					n, err := stdin.Write(p)
					if err != nil || n != len(p) || output.acknowledgeInput(n) != nil {
						stop()
						return
					}
				}
			}
		}()
	}

	//这里第一个协程获取用户的输入
	go func() {
		for {
			// p为用户输入
			messageType, p, err := ws.ReadMessage()
			if err != nil {
				stop()
				return
			}
			ws.SetReadDeadline(time.Now().Add(websocketReadTimeout))
			if output != nil && output.flow != nil && messageType == websocket.TextMessage && bytes.HasPrefix(p, []byte("\x00ack:")) {
				if n, err := strconv.Atoi(string(p[5:])); err == nil {
					output.flow.acknowledge(n)
				}
				continue
			}
			notifyActivity()
			if messageType == websocket.TextMessage && bytes.HasPrefix(p, []byte("resize:")) {
				resizeSlice := strings.Split(string(p), ":")
				if len(resizeSlice) != 3 {
					continue
				}
				rows, err := strconv.Atoi(strings.TrimSpace(resizeSlice[1]))
				if err != nil {
					log.Println("resize: invalid rows:", resizeSlice[1])
					continue
				}
				cols, err := strconv.Atoi(strings.TrimSpace(resizeSlice[2]))
				if err != nil {
					log.Println("resize: invalid cols:", resizeSlice[2])
					continue
				}
				if rows <= 0 || cols <= 0 {
					log.Printf("resize: invalid dimensions rows=%d cols=%d", rows, cols)
					continue
				}
				err = session.WindowChange(rows, cols)
				if err != nil {
					log.Printf("resize WindowChange failed rows=%d cols=%d: %v", rows, cols, err)
					stop()
					return
				}
				continue
			}
			if inputQueue != nil {
				select {
				case inputQueue <- p:
				case <-stopCh:
					return
				}
				continue
			}
			_, err = stdin.Write(p)
			if err != nil {
				stop()
				return
			}
		}
	}()

	go func() {
		if err := session.Wait(); err != nil {
			log.Println("ssh session wait:", err)
		}
		stop()
	}()

	defer func() {
		if output != nil && output.flow != nil {
			output.flow.close()
		}
		if output != nil {
			// A shell can exit before the 5 ms batching timer fires.
			output.Flush()
		}
		ws.Close()
		sclient.Close()

		if err := recover(); err != nil {
			log.Println("Connect panic recovered:", err)
		}
	}()
	// 设置ws超时时间timer
	stopTimer := time.NewTimer(timeout)
	defer stopTimer.Stop()
	wsPingTicker := time.NewTicker(websocketPingInterval)
	defer wsPingTicker.Stop()
	sshKeepaliveTicker := time.NewTicker(sshKeepaliveInterval)
	defer sshKeepaliveTicker.Stop()
	resetStopTimer := func() {
		if !stopTimer.Stop() {
			select {
			case <-stopTimer.C:
			default:
			}
		}
		stopTimer.Reset(timeout)
	}
	// 主循环
	for {
		select {
		case <-stopCh:
			return
		case <-activityCh:
			resetStopTimer()
		case <-wsPingTicker.C:
			if err := ws.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(websocketWriteTimeout)); err != nil {
				log.Println("websocket ping failed:", err)
				return
			}
		case <-sshKeepaliveTicker.C:
			if client != nil {
				if _, _, err := client.SendRequest("keepalive@webssh", false, nil); err != nil {
					log.Println("ssh keepalive failed:", err)
					return
				}
			}
		case <-stopTimer.C:
			tipMsg := []byte(fmt.Sprintf("\u001B[33m%s\u001B[0m", closeTip))
			if sclient.wsOutput != nil {
				sclient.wsOutput.Write(tipMsg)
				sclient.wsOutput.Flush()
			} else {
				ws.WriteMessage(websocket.TextMessage, tipMsg)
			}
			return
		}
	}
}
