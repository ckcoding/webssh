package controller

import (
	"log"
	"strings"
	"webssh/core"

	"github.com/gin-gonic/gin"
)

type sshSessionRequest struct {
	SSHInfo   string `json:"sshInfo"`
	Host      string `json:"host"`
	Username  string `json:"username"`
	Port      int    `json:"port"`
	Password  string `json:"password"`
	Logintype int    `json:"logintype"`
}

// CreateSSHSession 创建短期连接会话，避免后续 WebSocket/SSE/下载 URL 继续携带完整 sshInfo。
func CreateSSHSession(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	

	var req sshSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	var (
		client core.SSHClient
		err    error
	)
	if strings.TrimSpace(req.SSHInfo) != "" {
		client, err = core.DecodedMsgToSSHClient(req.SSHInfo)
	} else {
		client, err = core.BuildSSHClient(req.Host, req.Username, req.Password, req.Port, req.Logintype)
	}
	if err != nil {
		log.Println("ssh session:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	if strings.TrimSpace(client.IPAddress) == "" || strings.TrimSpace(client.Username) == "" || strings.TrimSpace(client.Password) == "" {
		responseBody.Msg = "invalid ssh session request"
		return &responseBody
	}

	session, err := core.CreateConnectionSession(client)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	responseBody.Data = map[string]interface{}{
		"sessionId": session.ID,
	}
	return &responseBody
}
