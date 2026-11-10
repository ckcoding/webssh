package core

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// WcList 保存当前上传任务的进度计数器，改用 sync.Map 提升并发查找性能。
var WcList sync.Map

// WriteCounter 记录单个上传任务已写入的字节数。
type WriteCounter struct {
	total    atomic.Int64
	Id       string
	progress chan struct{}
	closeMu  sync.Mutex
	closed   bool
}

// NewWriteCounter 创建带进度通知 channel 的计数器。
func NewWriteCounter(id string) *WriteCounter {
	return &WriteCounter{
		Id:       id,
		progress: make(chan struct{}, 1),
	}
}

// Progress 返回进度事件通道；计数器被移除时通道关闭。
func (wc *WriteCounter) Progress() <-chan struct{} {
	return wc.progress
}

func (wc *WriteCounter) closeProgress() {
	wc.closeMu.Lock()
	defer wc.closeMu.Unlock()
	if wc.closed || wc.progress == nil {
		return
	}
	wc.closed = true
	close(wc.progress)
}

func (wc *WriteCounter) TotalBytes() int64 {
	return wc.total.Load()
}

func AddWriteCounter(counter *WriteCounter) {
	WcList.Store(counter.Id, counter)
}

func RemoveWriteCounter(id string) {
	if val, ok := WcList.LoadAndDelete(id); ok {
		if counter, ok := val.(*WriteCounter); ok {
			counter.closeProgress()
		}
	}
}

// FindWriteCounter 按 id 查找当前活跃的计数器，找不到返回 nil。
func FindWriteCounter(id string) *WriteCounter {
	if val, ok := WcList.Load(id); ok {
		if counter, ok := val.(*WriteCounter); ok {
			return counter
		}
	}
	return nil
}

// Write 实现 io.Writer，用于累计已传输字节数。
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.total.Add(int64(n))
	if wc.progress != nil {
		select {
		case wc.progress <- struct{}{}:
		default:
		}
	}
	return n, nil
}

type wsOutput struct {
	ws         *websocket.Conn
	mu         *sync.Mutex
	activityFn func()
	buf        []byte
	flushTimer *time.Timer
	closed     bool
	err        error
	flow       *terminalFlow
}

const (
	wsOutputFlushInterval        = 5 * time.Millisecond
	wsOutputMaxBuffer            = 32 * 1024
	wsOutputInitialBuf           = 4 * 1024
	wsOutputCompressionThreshold = 1024
)

func (w *wsOutput) Write(p []byte) (int, error) {
	if w.flow != nil {
		// Limit each reservation, including writes larger than the output window.
		written := 0
		for len(p) > 0 {
			n := min(len(p), wsOutputMaxBuffer)
			if err := w.flow.reserve(n); err != nil {
				return written, err
			}
			if err := w.writeChunk(p[:n]); err != nil {
				return written, err
			}
			written += n
			p = p[n:]
		}
		return written, nil
	}
	return len(p), w.writeChunk(p)
}

func (w *wsOutput) writeChunk(p []byte) error {
	if w.mu != nil {
		w.mu.Lock()
		defer w.mu.Unlock()
	}
	if w.closed {
		return io.ErrClosedPipe
	}
	if w.err != nil {
		return w.err
	}
	w.buf = append(w.buf, p...)
	if len(w.buf) >= wsOutputMaxBuffer {
		return w.flushLocked()
	}
	if w.flushTimer == nil {
		w.flushTimer = time.AfterFunc(wsOutputFlushInterval, func() {
			if w.mu != nil {
				w.mu.Lock()
				defer w.mu.Unlock()
			}
			w.flushLocked()
		})
	}
	return nil
}

func (w *wsOutput) flushLocked() error {
	if w.flushTimer != nil {
		w.flushTimer.Stop()
		w.flushTimer = nil
	}
	if len(w.buf) == 0 || w.closed {
		return nil
	}
	w.ws.SetWriteDeadline(time.Now().Add(websocketWriteTimeout))
	w.ws.EnableWriteCompression(len(w.buf) >= wsOutputCompressionThreshold)
	err := w.ws.WriteMessage(websocket.BinaryMessage, w.buf)
	w.buf = w.buf[:0]
	if err == nil && w.activityFn != nil {
		w.activityFn()
	}
	if err != nil {
		w.err = err
		if w.flow != nil {
			w.flow.close()
		}
		// Wake the reader and release SSH resources after a failed timer flush.
		w.ws.Close()
	}
	return err
}

func (w *wsOutput) Flush() {
	if w.mu != nil {
		w.mu.Lock()
		defer w.mu.Unlock()
	}
	w.flushLocked()
}

func (w *wsOutput) Close() {
	if w.flow != nil {
		w.flow.close()
	}
	if w.mu != nil {
		w.mu.Lock()
		defer w.mu.Unlock()
	}
	w.flushLocked()
	w.closed = true
}

func (w *wsOutput) setActivityFn(fn func()) {
	w.mu.Lock()
	w.activityFn = fn
	w.mu.Unlock()
}

func (w *wsOutput) acknowledgeInput(n int) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ws.SetWriteDeadline(time.Now().Add(websocketWriteTimeout))
	w.ws.EnableWriteCompression(false)
	return w.ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\x00inputack:%d", n)))
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

// SSHDiskInfo 单个磁盘/挂载点信息
type SSHDiskInfo struct {
	Device      string  `json:"device"`
	Mount       string  `json:"mount"`
	TotalMB     int64   `json:"totalMB"`
	AvailableMB int64   `json:"availableMB"`
	UsedMB      int64   `json:"usedMB"`
	Usage       float64 `json:"usage"`
}

// SSHGPUInfo 单块显卡信息
type SSHGPUInfo struct {
	Name          string  `json:"name"`
	Driver        string  `json:"driver"`
	VRAMMB        int64   `json:"vramMB"`
	MemoryTotalMB int64   `json:"memoryTotalMB"`
	MemoryUsedMB  int64   `json:"memoryUsedMB"`
	MemoryUsage   float64 `json:"memoryUsage"`
	Usage         float64 `json:"usage"`
	Temperature   int     `json:"temperature"`
	Resolution    string  `json:"resolution"`
}

// SSHSystemInfo 远端系统信息
type SSHSystemInfo struct {
	ConnectAddress    string        `json:"connectAddress"`
	Hostname          string        `json:"hostname"`
	IP                string        `json:"ip"`
	CountryFlag       string        `json:"countryFlag"`
	Location          string        `json:"location"`
	OS                string        `json:"os"`
	Kernel            string        `json:"kernel"`
	LoadAverage       string        `json:"loadAverage"`
	CPUCores          int           `json:"cpuCores"`
	CPUUsage          float64       `json:"cpuUsage"`
	MemoryTotalMB     int64         `json:"memoryTotalMB"`
	MemoryAvailableMB int64         `json:"memoryAvailableMB"`
	MemoryUsedMB      int64         `json:"memoryUsedMB"`
	MemoryUsage       float64       `json:"memoryUsage"`
	DiskMount         string        `json:"diskMount"`
	DiskTotalMB       int64         `json:"diskTotalMB"`
	DiskAvailableMB   int64         `json:"diskAvailableMB"`
	DiskUsedMB        int64         `json:"diskUsedMB"`
	DiskUsage         float64       `json:"diskUsage"`
	Disks             []SSHDiskInfo `json:"disks"`
	GPUs              []SSHGPUInfo  `json:"gpus"`
	Uptime            string        `json:"uptime"`
	UpdatedAt         string        `json:"updatedAt"`
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
	defer func() {
		if err := recover(); err != nil {
			log.Println("SSHClient Close recover from panic: ", err)
		}
	}()

	if sclient.wsOutput != nil {
		sclient.wsOutput.Close()
		sclient.wsOutput = nil
	}
	if sclient.StdinPipe != nil {
		sclient.StdinPipe.Close()
		sclient.StdinPipe = nil
	}
	if sclient.Session != nil {
		sclient.Session.Close()
		sclient.Session = nil
	}
	if sclient.Sftp != nil {
		sclient.Sftp.Close()
		sclient.Sftp = nil
	}
	if sclient.Client != nil {
		sclient.Client.Close()
		sclient.Client = nil
	}
}
