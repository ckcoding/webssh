package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"webssh/core"

	"github.com/gin-gonic/gin"
)

const (
	systemInfoStreamInterval = 5 * time.Second
	systemInfoKeepaliveEvery = 15 * time.Second
)

func collectSSHSystemInfo(sshInfo string) (core.SSHSystemInfo, error) {
	session, err := core.GetCommandSession(sshInfo)
	if err != nil {
		return core.SSHSystemInfo{}, err
	}

	var info core.SSHSystemInfo
	err = session.Use(func(sshClient *core.SSHClient) error {
		var collectErr error
		info, collectErr = core.CollectSSHSystemInfo(sshClient)
		return collectErr
	})
	if err != nil {
		return core.SSHSystemInfo{}, err
	}

	return info, nil
}

func writeSSEData(c *gin.Context, flusher http.Flusher, payload interface{}) error {
	content, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", content); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

// GetSSHSystemInfo 获取远端服务器系统信息
func GetSSHSystemInfo(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)

	sshInfo := c.DefaultQuery("sshInfo", "")
	info, err := collectSSHSystemInfo(sshInfo)
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}

	responseBody.Data = map[string]interface{}{
		"info": info,
	}
	return &responseBody
}

// StreamSSHSystemInfo 通过 SSE 持续推送远端服务器动态信息。
func StreamSSHSystemInfo(c *gin.Context) {
	sshInfo := c.DefaultQuery("sshInfo", "")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"Msg": "stream not supported"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	if _, err := fmt.Fprint(c.Writer, "retry: 5000\n\n"); err != nil {
		return
	}
	flusher.Flush()

	sendSnapshot := func() error {
		info, err := collectSSHSystemInfo(sshInfo)
		if err != nil {
			fmt.Println(err)
			return writeSSEData(c, flusher, ResponseBody{Msg: err.Error()})
		}
		return writeSSEData(c, flusher, ResponseBody{
			Msg: "success",
			Data: map[string]interface{}{
				"info": info,
			},
		})
	}

	if err := sendSnapshot(); err != nil {
		return
	}

	ticker := time.NewTicker(systemInfoStreamInterval)
	defer ticker.Stop()
	keepalive := time.NewTicker(systemInfoKeepaliveEvery)
	defer keepalive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			if err := sendSnapshot(); err != nil {
				return
			}
		case <-keepalive.C:
			// SSE 注释行，客户端忽略，只为维持反向代理和 HTTP/1.1 连接不被缓冲/超时。
			if _, err := fmt.Fprint(c.Writer, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
