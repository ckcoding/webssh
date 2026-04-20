package controller

import (
	"fmt"
	"time"
	"webssh/core"

	"github.com/gin-gonic/gin"
)

type sshCheckRequest struct {
	Host      string `json:"host"`
	Username  string `json:"username"`
	Port      int    `json:"port"`
	Password  string `json:"password"`
	Logintype int    `json:"logintype"`
	ProbeOnly bool   `json:"probeOnly"`
}

// CheckSSHConnection 预连接检测 SSH 是否可达且认证可用。
func CheckSSHConnection(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)

	var req sshCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}

	sshClient, err := core.BuildSSHClient(req.Host, req.Username, req.Password, req.Port, req.Logintype)
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}

	startedAt := time.Now()
	if err := sshClient.GenerateClient(); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	defer sshClient.Close()

	latencyMs := time.Since(startedAt).Milliseconds()
	if latencyMs < 1 {
		latencyMs = 1
	}

	responseBody.Data = map[string]interface{}{
		"latencyMs": latencyMs,
	}
	if req.ProbeOnly {
		return &responseBody
	}

	info, err := core.CollectSSHSystemInfo(&sshClient)
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}

	responseBody.Data = map[string]interface{}{
		"latencyMs": latencyMs,
		"info":      info,
	}
	return &responseBody
}
