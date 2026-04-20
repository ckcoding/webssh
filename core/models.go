package core

import (
	"github.com/gorilla/websocket"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"sync"
	"unicode/utf8"
)

// WcList 保存当前上传任务的进度计数器。
var WcList []*WriteCounter
var wcMu sync.RWMutex

// WriteCounter 记录单个上传任务已写入的字节数。
type WriteCounter struct {
	Total int
	Id    string
}

func AddWriteCounter(counter *WriteCounter) {
	wcMu.Lock()
	defer wcMu.Unlock()
	WcList = append(WcList, counter)
}

func RemoveWriteCounter(id string) {
	wcMu.Lock()
	defer wcMu.Unlock()
	if len(WcList) < 2 {
		WcList = nil
		return
	}
	for i := 0; i < len(WcList); i++ {
		if WcList[i].Id == id {
			WcList = append(WcList[:i], WcList[i+1:]...)
			break
		}
	}
}

func SnapshotWriteCounters() []*WriteCounter {
	wcMu.RLock()
	defer wcMu.RUnlock()
	if len(WcList) == 0 {
		return nil
	}
	snapshot := make([]*WriteCounter, len(WcList))
	copy(snapshot, WcList)
	return snapshot
}

// Write 实现 io.Writer，用于累计已传输字节数。
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += n
	return n, nil
}

type wsOutput struct {
	ws         *websocket.Conn
	mu         *sync.Mutex
	activityFn func()
}

// Write 实现 io.Writer，将 SSH 输出写入 websocket。
func (w *wsOutput) Write(p []byte) (int, error) {
	// 将非法 UTF-8 字节替换为占位字符，避免 websocket 文本帧写入失败。
	if !utf8.Valid(p) {
		bufStr := string(p)
		buf := make([]rune, 0, len(bufStr))
		for _, r := range bufStr {
			if r == utf8.RuneError {
				buf = append(buf, []rune("@")...)
			} else {
				buf = append(buf, r)
			}
		}
		p = []byte(string(buf))
	}
	if w.mu != nil {
		w.mu.Lock()
		defer w.mu.Unlock()
	}
	err := w.ws.WriteMessage(websocket.TextMessage, p)
	if err == nil && w.activityFn != nil {
		w.activityFn()
	}
	return len(p), err
}

// SSHClient 描述一次 SSH 连接所需的上下文。
type SSHClient struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	IPAddress string `json:"ipaddress"`
	Port      int    `json:"port"`
	LoginType int    `json:"logintype"`
	Client    *ssh.Client
	Sftp      *sftp.Client
	StdinPipe io.WriteCloser
	Session   *ssh.Session
	wsWriteMu *sync.Mutex
	wsOutput  *wsOutput
}

// SSHSystemInfo 远端系统信息
type SSHSystemInfo struct {
	ConnectAddress    string  `json:"connectAddress"`
	Hostname          string  `json:"hostname"`
	IP                string  `json:"ip"`
	CountryFlag       string  `json:"countryFlag"`
	Location          string  `json:"location"`
	OS                string  `json:"os"`
	Kernel            string  `json:"kernel"`
	LoadAverage       string  `json:"loadAverage"`
	CPUCores          int     `json:"cpuCores"`
	CPUUsage          float64 `json:"cpuUsage"`
	MemoryTotalMB     int64   `json:"memoryTotalMB"`
	MemoryAvailableMB int64   `json:"memoryAvailableMB"`
	MemoryUsedMB      int64   `json:"memoryUsedMB"`
	MemoryUsage       float64 `json:"memoryUsage"`
	DiskMount         string  `json:"diskMount"`
	DiskTotalMB       int64   `json:"diskTotalMB"`
	DiskAvailableMB   int64   `json:"diskAvailableMB"`
	DiskUsedMB        int64   `json:"diskUsedMB"`
	DiskUsage         float64 `json:"diskUsage"`
	Uptime            string  `json:"uptime"`
	UpdatedAt         string  `json:"updatedAt"`
}

// NewSSHClient 返回默认 SSH 配置。
func NewSSHClient() SSHClient {
	return SSHClient{
		Username: "root",
		Port:     22,
	}
}

// Close 关闭 SSHClient 中已创建的所有可关闭资源。
func (sclient *SSHClient) Close() {
	defer func() { // 防止关闭阶段的异常影响主流程。
		if err := recover(); err != nil {
			log.Println("SSHClient Close recover from panic: ", err)
		}
	}()

	if sclient.StdinPipe != nil {
		sclient.StdinPipe.Close()
		sclient.StdinPipe = nil
	}
	if sclient.Session != nil {
		sclient.Session.Close()
		sclient.Session = nil
	}
	sclient.wsOutput = nil
	if sclient.Sftp != nil {
		sclient.Sftp.Close()
		sclient.Sftp = nil
	}
	if sclient.Client != nil {
		sclient.Client.Close()
		sclient.Client = nil
	}
}
