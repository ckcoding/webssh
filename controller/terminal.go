package controller

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"webssh/core"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  32768,
	WriteBufferSize: 32768,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		originURL, err := url.Parse(origin)
		if err != nil {
			return false
		}
		if sameRequestHost(originURL.Host, r.Host) {
			return true
		}
		return sameRequestHost(originURL.Host, firstForwardedHost(r.Header.Get("X-Forwarded-Host")))
	},
	EnableCompression: true,
}

func firstForwardedHost(value string) string {
	if value == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(value, ",")[0])
}

func sameRequestHost(originHost, requestHost string) bool {
	originHost = strings.TrimSpace(originHost)
	requestHost = strings.TrimSpace(requestHost)
	if originHost == "" || requestHost == "" {
		return false
	}
	if strings.EqualFold(originHost, requestHost) {
		return true
	}
	return strings.EqualFold(hostnameOnly(originHost), hostnameOnly(requestHost))
}

func hostnameOnly(value string) string {
	if value == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(host, "[]")
	}
	if parsed := strings.Trim(value, "[]"); strings.Count(parsed, ":") == 0 {
		return parsed
	}
	return value
}

// TermWs 获取终端ws
func TermWs(c *gin.Context, timeout time.Duration) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}

	sshInfo := c.DefaultQuery("sshInfo", "")
	sessionID := c.DefaultQuery("sessionId", "")
	cols := c.DefaultQuery("cols", "150")
	rows := c.DefaultQuery("rows", "35")
	closeTip := c.DefaultQuery("closeTip", "Connection timed out!")
	col, _ := strconv.Atoi(cols)
	row, _ := strconv.Atoi(rows)
	sshClient, _, err := core.ResolveSSHClient(sessionID, sshInfo)
	if err != nil {
		log.Println("resolve ssh client:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("websocket upgrade:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	wsConn.EnableWriteCompression(true)
	wsConn.SetCompressionLevel(1)
	if prepared := core.TakePreparedTerminalConnection(sessionID); prepared != nil {
		sshClient = *prepared
	} else {
		err = sshClient.GenerateClient()
	}
	if err != nil {
		wsConn.WriteMessage(1, []byte(err.Error()))
		wsConn.Close()
		log.Println("ssh generate client:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	if err := sshClient.InitTerminal(wsConn, row, col, c.Query("flowControl") == "1"); err != nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		wsConn.Close()
		sshClient.Close()
		log.Println("ssh init terminal:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	sshClient.Connect(wsConn, timeout, closeTip)
	return &responseBody
}
