package controller

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"webssh/core"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// File 表示远端文件列表中的单个条目。
type File struct {
	Name       string
	Size       string
	ModifyTime string
	IsDir      bool
}

type createEntryRequest struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	SshInfo   string `json:"sshInfo"`
	SessionID string `json:"sessionId"`
}

type renameEntryRequest struct {
	Path      string `json:"path"`
	NewName   string `json:"newName"`
	SshInfo   string `json:"sshInfo"`
	SessionID string `json:"sessionId"`
}

func detectInlineContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			return contentType
		}
	}
	switch ext {
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".mkv":
		return "video/x-matroska"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".flac":
		return "audio/flac"
	case ".aac":
		return "audio/aac"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

const (
	// BYTE 字节
	BYTE = 1 << (10 * iota)
	// KILOBYTE 千字节
	KILOBYTE
	// MEGABYTE 兆字节
	MEGABYTE
	// GIGABYTE 吉字节
	GIGABYTE
	// TERABYTE 太字节
	TERABYTE
	// PETABYTE 拍字节
	PETABYTE
	// EXABYTE 艾字节
	EXABYTE
)

const defaultMaxEditFileBytes int64 = 5 * 1024 * 1024

func maxEditFileBytes() int64 {
	raw := strings.TrimSpace(os.Getenv("WEBSSH_MAX_EDIT_FILE_BYTES"))
	if raw == "" {
		return defaultMaxEditFileBytes
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		return defaultMaxEditFileBytes
	}
	return parsed
}

func ensureEditableContentSize(size int64) error {
	limit := maxEditFileBytes()
	if size > limit {
		return fmt.Errorf("file is too large to edit (max %s)", Bytefmt(uint64(limit)))
	}
	return nil
}

// Bytefmt 将字节数格式化为便于展示的单位字符串。
func Bytefmt(bytes uint64) string {
	unit := ""
	value := float64(bytes)

	switch {
	case bytes >= EXABYTE:
		unit = "E"
		value = value / EXABYTE
	case bytes >= PETABYTE:
		unit = "P"
		value = value / PETABYTE
	case bytes >= TERABYTE:
		unit = "T"
		value = value / TERABYTE
	case bytes >= GIGABYTE:
		unit = "G"
		value = value / GIGABYTE
	case bytes >= MEGABYTE:
		unit = "M"
		value = value / MEGABYTE
	case bytes >= KILOBYTE:
		unit = "K"
		value = value / KILOBYTE
	case bytes >= BYTE:
		unit = "B"
	case bytes == 0:
		return "0B"
	}

	result := strconv.FormatFloat(value, 'f', 2, 64)
	result = strings.TrimSuffix(result, ".00")
	return result + unit
}

type fileEntries []File

// Len 返回元素个数。
func (f fileEntries) Len() int { return len(f) }

// Swap 交换两个元素的位置。
func (f fileEntries) Swap(i, j int) { f[i], f[j] = f[j], f[i] }

// Less 让目录排在文件前面，再按名称排序。
func (f fileEntries) Less(i, j int) bool {
	if f[i].IsDir && !f[j].IsDir {
		return true
	}
	if !f[i].IsDir && f[j].IsDir {
		return false
	}
	return strings.ToLower(f[i].Name) < strings.ToLower(f[j].Name)
}

func normalizeRemotePath(raw string) string {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "/"
	}
	path = strings.ReplaceAll(path, "\\", "/")
	path = pathpkg.Clean(path)
	if path == "." {
		return "/"
	}
	return path
}

func sanitizeCreateName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	switch {
	case name == "":
		return "", fmt.Errorf("name is empty")
	case name == "." || name == "..":
		return "", fmt.Errorf("invalid name")
	case strings.Contains(name, "/") || strings.Contains(name, "\\"):
		return "", fmt.Errorf("name cannot contain path separators")
	case strings.ContainsRune(name, '\x00'):
		return "", fmt.Errorf("invalid name")
	default:
		return name, nil
	}
}

func buildCreateTargetPath(dirPath, name string) (string, error) {
	cleanName, err := sanitizeCreateName(name)
	if err != nil {
		return "", err
	}
	return pathpkg.Join(normalizeRemotePath(dirPath), cleanName), nil
}

func sanitizeRelativeDir(raw string) (string, error) {
	dir := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if dir == "" {
		return "", nil
	}
	parts := strings.Split(dir, "/")
	cleanParts := make([]string, 0, len(parts))
	for _, part := range parts {
		cleanPart, err := sanitizeCreateName(part)
		if err != nil {
			return "", err
		}
		cleanParts = append(cleanParts, cleanPart)
	}
	return pathpkg.Join(cleanParts...), nil
}

func buildUploadTargetPath(basePath, dir, filename string) (string, string, error) {
	cleanFilename, err := sanitizeCreateName(filename)
	if err != nil {
		return "", "", err
	}
	cleanDir, err := sanitizeRelativeDir(dir)
	if err != nil {
		return "", "", err
	}
	targetDir := normalizeRemotePath(basePath)
	if cleanDir != "" {
		targetDir = pathpkg.Join(targetDir, cleanDir)
	}
	return pathpkg.Join(targetDir, cleanFilename), targetDir, nil
}

// UploadFile 上传文件
func UploadFile(c *gin.Context) *ResponseBody {
	var err error
	responseBody := ResponseBody{Msg: "success"}
	
	sshInfo := c.PostForm("sshInfo")
	sessionID := c.PostForm("sessionId")
	id := c.PostForm("id")
	session, err := core.GetFileSessionByID(sessionID, sshInfo)
	if err != nil {
		log.Println("file session:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	defer file.Close()
	path := c.DefaultPostForm("path", "/root")
	dir := c.DefaultPostForm("dir", "")
	targetPath, targetDir, err := buildUploadTargetPath(path, dir, header.Filename)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	err = session.Use(func(sshClient *core.SSHClient) error {
		if dir != "" {
			if err := sshClient.Mkdirs(targetDir); err != nil {
				return err
			}
		}
		return sshClient.Upload(file, id, targetPath)
	})
	if err != nil {
		log.Println("upload file:", err)
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

// DownloadFile 下载文件
func DownloadFile(c *gin.Context) *ResponseBody {
	var err error
	responseBody := ResponseBody{Msg: "success"}
	
	path := strings.TrimSpace(c.DefaultQuery("path", "/root"))
	inline := c.DefaultQuery("inline", "") == "1"
	sshInfo := c.DefaultQuery("sshInfo", "")
	sessionID := c.DefaultQuery("sessionId", "")
	session, err := core.GetFileSessionByID(sessionID, sshInfo)
	if err != nil {
		log.Println("download file session:", err)
		responseBody.Msg = err.Error()
		if !c.Writer.Written() {
			c.JSON(http.StatusBadRequest, responseBody)
		}
		return &responseBody
	}
	err = session.Use(func(sshClient *core.SSHClient) error {
		sftpFile, err := sshClient.Download(path)
		if err != nil {
			return err
		}
		defer sftpFile.Close()
		disposition := "attachment"
		if inline {
			disposition = "inline"
			c.Header("Content-Type", detectInlineContentType(path))
		}
		c.Header("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{
			"filename": filepath.Base(path),
		}))
		c.Writer.WriteHeader(http.StatusOK)
		_, err = io.Copy(c.Writer, sftpFile)
		return err
	})
	if err != nil {
		log.Println("download file:", err)
		responseBody.Msg = err.Error()
		if !c.Writer.Written() {
			c.JSON(http.StatusBadRequest, responseBody)
		}
	}
	return &responseBody
}

// UploadProgressWs 获取上传进度ws
func UploadProgressWs(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("upload progress ws upgrade:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	defer wsConn.Close()
	id := strings.TrimSpace(c.Query("id"))
	if id == "" {
		return &responseBody
	}

	// 监听客户端断开：任何 read 错误（关闭、ping 失败）都会触发退出。
	clientClosed := make(chan struct{})
	go func() {
		defer close(clientClosed)
		for {
			if _, _, err := wsConn.NextReader(); err != nil {
				return
			}
		}
	}()

	// 上传协程可能稍晚注册计数器；慢网络下浏览器请求体到达完成后，
	// 后端才会开始 SFTP 写入，所以这里需要给大文件和弱网更充足的等待窗口。
	var counter *core.WriteCounter
	waitDeadline := time.Now().Add(15 * time.Second)
	for counter == nil {
		counter = core.FindWriteCounter(id)
		if counter != nil {
			break
		}
		if time.Now().After(waitDeadline) {
			return &responseBody
		}
		select {
		case <-clientClosed:
			return &responseBody
		case <-c.Request.Context().Done():
			return &responseBody
		case <-time.After(50 * time.Millisecond):
		}
	}

	writeTotal := func() error {
		_ = wsConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		return wsConn.WriteMessage(websocket.TextMessage, []byte(strconv.FormatInt(counter.TotalBytes(), 10)))
	}

	if err := writeTotal(); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	// 最多 1 秒发一次心跳，防止静默连接被代理关闭。
	heartbeat := time.NewTicker(1 * time.Second)
	defer heartbeat.Stop()

	progress := counter.Progress()
	for {
		select {
		case <-clientClosed:
			return &responseBody
		case <-c.Request.Context().Done():
			return &responseBody
		case _, ok := <-progress:
			if err := writeTotal(); err != nil {
				return &responseBody
			}
			if !ok {
				// 计数器被移除，上传已收尾，写一次最终值后退出。
				return &responseBody
			}
		case <-heartbeat.C:
			if err := writeTotal(); err != nil {
				return &responseBody
			}
		}
	}
}

// FileList 获取文件列表
func FileList(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	
	path := c.DefaultQuery("path", "/root")
	sshInfo := c.DefaultQuery("sshInfo", "")
	sessionID := c.DefaultQuery("sessionId", "")
	session, err := core.GetFileSessionByID(sessionID, sshInfo)
	if err != nil {
		log.Println("file list session:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	var (
		fileList fileEntries
		fileSize string
	)
	err = session.Use(func(sshClient *core.SSHClient) error {
		files, err := sshClient.Sftp.ReadDir(path)
		if err != nil {
			return err
		}
		for _, mFile := range files {
			if mFile.IsDir() {
				fileSize = "-"
			} else {
				fileSize = Bytefmt(uint64(mFile.Size()))
			}
			file := File{Name: mFile.Name(), IsDir: mFile.IsDir(), Size: fileSize, ModifyTime: mFile.ModTime().Format("2006-01-02 15:04:05")}
			fileList = append(fileList, file)
		}
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "exist") {
			responseBody.Msg = fmt.Sprintf("Directory %s: no such file or directory", path)
		} else {
			responseBody.Msg = err.Error()
		}
		return &responseBody
	}
	sort.Stable(fileList)
	responseBody.Data = map[string]interface{}{
		"path": path,
		"list": fileList,
	}
	return &responseBody
}

// GetFileContent 获取文件内容
func GetFileContent(c *gin.Context) *ResponseBody {
	var err error
	responseBody := ResponseBody{Msg: "success"}
	
	path := strings.TrimSpace(c.DefaultQuery("path", ""))
	sshInfo := c.DefaultQuery("sshInfo", "")
	sessionID := c.DefaultQuery("sessionId", "")
	if path == "" {
		responseBody.Msg = "path is empty"
		return &responseBody
	}
	session, err := core.GetFileSessionByID(sessionID, sshInfo)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	err = session.Use(func(sshClient *core.SSHClient) error {
		file, err := sshClient.Sftp.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("path is a directory")
		}
		if err := ensureEditableContentSize(info.Size()); err != nil {
			return err
		}
		limit := maxEditFileBytes()
		content, err := io.ReadAll(&io.LimitedReader{R: file, N: limit + 1})
		if err != nil {
			return err
		}
		if err := ensureEditableContentSize(int64(len(content))); err != nil {
			return err
		}
		responseBody.Data = string(content)
		return nil
	})
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	return &responseBody
}

// SaveFileContent 保存文件内容
func SaveFileContent(c *gin.Context) *ResponseBody {
	var (
		err error
		req struct {
			Path      string `json:"path"`
			Content   string `json:"content"`
			SshInfo   string `json:"sshInfo"`
			SessionID string `json:"sessionId"`
		}
	)
	responseBody := ResponseBody{Msg: "success"}
	

	if err := c.ShouldBindJSON(&req); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	if req.Path == "" {
		responseBody.Msg = "path is empty"
		return &responseBody
	}
	if err := ensureEditableContentSize(int64(len(req.Content))); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	session, err := core.GetFileSessionByID(req.SessionID, req.SshInfo)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	err = session.Use(func(sshClient *core.SSHClient) error {
		file, err := sshClient.Sftp.Create(req.Path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = file.Write([]byte(req.Content))
		return err
	})
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	return &responseBody
}

// CreateFile 创建空文件
func CreateFile(c *gin.Context) *ResponseBody {
	var (
		err error
		req createEntryRequest
	)
	responseBody := ResponseBody{Msg: "success"}
	

	if err := c.ShouldBindJSON(&req); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	targetPath, err := buildCreateTargetPath(req.Path, req.Name)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	session, err := core.GetFileSessionByID(req.SessionID, req.SshInfo)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	err = session.Use(func(sshClient *core.SSHClient) error {
		return sshClient.CreateEmptyFile(targetPath)
	})
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	responseBody.Data = map[string]interface{}{
		"path": targetPath,
	}
	return &responseBody
}

// CreateDirectory 创建文件夹
func CreateDirectory(c *gin.Context) *ResponseBody {
	var (
		err error
		req createEntryRequest
	)
	responseBody := ResponseBody{Msg: "success"}
	

	if err := c.ShouldBindJSON(&req); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	targetPath, err := buildCreateTargetPath(req.Path, req.Name)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	session, err := core.GetFileSessionByID(req.SessionID, req.SshInfo)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	err = session.Use(func(sshClient *core.SSHClient) error {
		return sshClient.Mkdir(targetPath)
	})
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	responseBody.Data = map[string]interface{}{
		"path": targetPath,
	}
	return &responseBody
}

// RenameEntry 重命名文件或目录（同级重命名，不跨目录）
func RenameEntry(c *gin.Context) *ResponseBody {
	var (
		err error
		req renameEntryRequest
	)
	responseBody := ResponseBody{Msg: "success"}
	

	if err := c.ShouldBindJSON(&req); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	oldPath := normalizeRemotePath(req.Path)
	if oldPath == "" || oldPath == "/" {
		responseBody.Msg = "invalid source path"
		return &responseBody
	}

	cleanName, err := sanitizeCreateName(req.NewName)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	parentDir := pathpkg.Dir(oldPath)
	if parentDir == "" {
		parentDir = "/"
	}
	newPath := pathpkg.Join(parentDir, cleanName)
	if newPath == oldPath {
		responseBody.Msg = "new name is the same as the current name"
		return &responseBody
	}

	session, err := core.GetFileSessionByID(req.SessionID, req.SshInfo)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	err = session.Use(func(sshClient *core.SSHClient) error {
		return sshClient.Rename(oldPath, newPath)
	})
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	responseBody.Data = map[string]interface{}{
		"oldPath": oldPath,
		"path":    newPath,
	}
	return &responseBody
}
