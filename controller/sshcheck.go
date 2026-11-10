package controller

import (
	"log"
	"strings"
	"time"
	"webssh/core"

	"github.com/gin-gonic/gin"
)

type sshCheckRequest struct {
	Host           string `json:"host"`
	Username       string `json:"username"`
	Port           int    `json:"port"`
	Password       string `json:"password"`
	Logintype      int    `json:"logintype"`
	ProbeOnly      bool   `json:"probeOnly"`
	PrepareSession bool   `json:"prepareSession"`
}

// CheckSSHConnection 预连接检测 SSH 是否可达且认证可用。
func CheckSSHConnection(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}

	var req sshCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println("ssh check bind:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}

	sshClient, err := core.BuildSSHClient(req.Host, req.Username, req.Password, req.Port, req.Logintype)
	if err != nil {
		log.Println("ssh check build client:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}

	startedAt := time.Now()
	if err := sshClient.GenerateClient(); err != nil {
		log.Printf("ssh check connect failed: host=%s port=%d user=%s logintype=%d err=%v", req.Host, req.Port, req.Username, req.Logintype, err)
		if err.Error() == "EOF" || strings.Contains(err.Error(), "EOF") {
			responseBody.Msg = "认证失败或连接被服务端意外关闭(EOF)，请检查密码或重试"
		} else {
			responseBody.Msg = err.Error()
		}
		return &responseBody
	}
	handedOff := false
	defer func() {
		if !handedOff {
			sshClient.Close()
		}
	}()

	latencyMs := time.Since(startedAt).Milliseconds()
	if latencyMs < 1 {
		latencyMs = 1
	}

	responseBody.Data = map[string]interface{}{
		"latencyMs": latencyMs,
	}
	if req.PrepareSession {
		if c.Request.Context().Err() != nil {
			responseBody.Msg = "connection request canceled"
			return &responseBody
		}
		session, err := core.PrepareTerminalConnection(sshClient)
		if err != nil {
			responseBody.Msg = err.Error()
			return &responseBody
		}
		handedOff = true
		responseBody.Data = map[string]interface{}{
			"latencyMs": latencyMs,
			"sessionId": session.ID,
		}
		return &responseBody
	}
	if req.ProbeOnly {
		return &responseBody
	}

	info, err := core.CollectSSHSystemInfo(&sshClient)
	if err != nil {
		log.Printf("ssh check collect system info warning: %v", err)
		// 即使采集系统信息失败，SSH连接本身是成功的，不应该阻断用户使用
		return &responseBody
	}

	responseBody.Data = map[string]interface{}{
		"latencyMs": latencyMs,
		"info":      info,
	}
	return &responseBody
}
