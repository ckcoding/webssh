package core

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
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
		client.Port = port
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
	return client, nil
}

// GenerateClient 创建ssh客户端
func (sclient *SSHClient) GenerateClient() error {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		config       ssh.Config
		err          error
	)
	auth = make([]ssh.AuthMethod, 0)
	if sclient.LoginType == 0 {
		auth = append(auth, ssh.Password(sclient.Password))
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
		},
		KeyExchanges: []string{
			"curve25519-sha256",
			"curve25519-sha256@libssh.org",
			"ecdh-sha2-nistp256",
			"ecdh-sha2-nistp384",
			"ecdh-sha2-nistp521",
			"diffie-hellman-group-exchange-sha256",
			"diffie-hellman-group14-sha256",
		},
	}
	clientConfig = &ssh.ClientConfig{
		User:    sclient.Username,
		Auth:    auth,
		Timeout: defaultSSHConnectTimeout,
		Config:  config,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	addr = fmt.Sprintf("%s:%d", sclient.IPAddress, sclient.Port)
	dialer := &net.Dialer{
		Timeout:   defaultSSHConnectTimeout,
		KeepAlive: tcpKeepaliveInterval,
	}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, clientConfig)
	if err != nil {
		conn.Close()
		return err
	}
	client = ssh.NewClient(sshConn, chans, reqs)
	sclient.Client = client
	return nil
}

// InitTerminal 初始化终端
func (sclient *SSHClient) InitTerminal(ws *websocket.Conn, rows, cols int) *SSHClient {
	sshSession, err := sclient.Client.NewSession()
	if err != nil {
		log.Println(err)
		return nil
	}
	sclient.Session = sshSession
	sclient.StdinPipe, _ = sshSession.StdinPipe()
	wsOutput := new(wsOutput)
	// 终端标准输出和标准错误都写入同一个 websocket 输出器。
	sshSession.Stdout = wsOutput
	sshSession.Stderr = wsOutput
	wsOutput.ws = ws
	sclient.wsWriteMu = &sync.Mutex{}
	wsOutput.mu = sclient.wsWriteMu
	sclient.wsOutput = wsOutput
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}

	if err := sshSession.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		return nil
	}
	if err := sshSession.Shell(); err != nil {
		return nil
	}
	return sclient
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

	if err := session.Run(command); err != nil {
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
	if sclient.wsOutput != nil {
		sclient.wsOutput.activityFn = notifyActivity
	}

	//这里第一个协程获取用户的输入
	go func() {
		for {
			// p为用户输入
			_, p, err := ws.ReadMessage()
			if err != nil {
				stop()
				return
			}
			ws.SetReadDeadline(time.Now().Add(websocketReadTimeout))
			notifyActivity()
			if strings.HasPrefix(string(p), "resize:") {
				resizeSlice := strings.Split(string(p), ":")
				if len(resizeSlice) != 3 {
					continue
				}
				rows, _ := strconv.Atoi(resizeSlice[1])
				cols, _ := strconv.Atoi(resizeSlice[2])
				err := sclient.Session.WindowChange(rows, cols)
				if err != nil {
					log.Println(err)
					stop()
					return
				}
				continue
			}
			_, err = sclient.StdinPipe.Write(p)
			if err != nil {
				stop()
				return
			}
		}
	}()

	go func() {
		if sclient.Session == nil {
			return
		}
		if err := sclient.Session.Wait(); err != nil {
			log.Println(err)
		}
		stop()
	}()

	defer func() {
		ws.Close()
		sclient.Close()

		if err := recover(); err != nil {
			log.Println(err)
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
				log.Println(err)
				return
			}
		case <-sshKeepaliveTicker.C:
			if sclient.Client != nil {
				if _, _, err := sclient.Client.SendRequest("keepalive@webssh", false, nil); err != nil {
					log.Println(err)
					return
				}
			}
		case <-stopTimer.C:
			if sclient.wsWriteMu != nil {
				sclient.wsWriteMu.Lock()
				ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\u001B[33m%s\u001B[0m", closeTip)))
				sclient.wsWriteMu.Unlock()
			} else {
				ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\u001B[33m%s\u001B[0m", closeTip)))
			}
			return
		}
	}
}
