package controller

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"webssh/core"

	"github.com/gin-gonic/gin"
)

// File 表示远端文件列表中的单个条目。
type File struct {
	Name       string
	Size       string
	ModifyTime string
	IsDir      bool
}

type createEntryRequest struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	SshInfo string `json:"sshInfo"`
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

// UploadFile 上传文件
func UploadFile(c *gin.Context) *ResponseBody {
	var err error
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	sshInfo := c.PostForm("sshInfo")
	id := c.PostForm("id")
	session, err := core.GetFileSession(sshInfo)
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	defer file.Close()
	path := strings.TrimSpace(c.DefaultPostForm("path", "/root"))
	pathArr := []string{strings.TrimRight(path, "/")}
	dir := c.DefaultPostForm("dir", "")
	if dir != "" {
		pathArr = append(pathArr, dir)
	}
	pathArr = append(pathArr, header.Filename)
	err = session.Use(func(sshClient *core.SSHClient) error {
		if dir != "" {
			if err := sshClient.Mkdirs(strings.Join(pathArr[:len(pathArr)-1], "/")); err != nil {
				return err
			}
		}
		return sshClient.Upload(file, id, strings.Join(pathArr, "/"))
	})
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

// DownloadFile 下载文件
func DownloadFile(c *gin.Context) *ResponseBody {
	var err error
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	path := strings.TrimSpace(c.DefaultQuery("path", "/root"))
	inline := c.DefaultQuery("inline", "") == "1"
	sshInfo := c.DefaultQuery("sshInfo", "")
	session, err := core.GetFileSession(sshInfo)
	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
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
	defer TimeCost(time.Now(), &responseBody)
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	defer wsConn.Close()
	id := c.Query("id")

	var ready, find bool
	for {
		counters := core.SnapshotWriteCounters()
		if !ready && len(counters) == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		for _, v := range counters {
			if v.Id == id {
				if err := wsConn.WriteMessage(1, []byte(strconv.Itoa(v.Total))); err != nil {
					responseBody.Msg = err.Error()
					return &responseBody
				}
				find = true
				if !ready {
					ready = true
				}
				break
			}
		}
		if ready && !find {
			break
		}

		if ready {
			time.Sleep(300 * time.Millisecond)
			find = false
		}
	}
	return &responseBody
}

// FileList 获取文件列表
func FileList(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	path := c.DefaultQuery("path", "/root")
	sshInfo := c.DefaultQuery("sshInfo", "")
	session, err := core.GetFileSession(sshInfo)
	if err != nil {
		fmt.Println(err)
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
	defer TimeCost(time.Now(), &responseBody)
	path := strings.TrimSpace(c.DefaultQuery("path", ""))
	sshInfo := c.DefaultQuery("sshInfo", "")
	if path == "" {
		responseBody.Msg = "path is empty"
		return &responseBody
	}
	session, err := core.GetFileSession(sshInfo)
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
		content, err := io.ReadAll(file)
		if err != nil {
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
			Path    string `json:"path"`
			Content string `json:"content"`
			SshInfo string `json:"sshInfo"`
		}
	)
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)

	if err := c.ShouldBindJSON(&req); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	if req.Path == "" {
		responseBody.Msg = "path is empty"
		return &responseBody
	}

	session, err := core.GetFileSession(req.SshInfo)
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
	defer TimeCost(time.Now(), &responseBody)

	if err := c.ShouldBindJSON(&req); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	targetPath, err := buildCreateTargetPath(req.Path, req.Name)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	session, err := core.GetFileSession(req.SshInfo)
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
	defer TimeCost(time.Now(), &responseBody)

	if err := c.ShouldBindJSON(&req); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	targetPath, err := buildCreateTargetPath(req.Path, req.Name)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	session, err := core.GetFileSession(req.SshInfo)
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
