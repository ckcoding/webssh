package controller

import (
	"context"
	"log"
	"time"
	"webssh/core"

	"github.com/gin-gonic/gin"
)

const overviewRequestTimeout = 20 * time.Second

// GetServerOverview 返回所有已保存主机的实时状态（无需被控端安装 Agent）。
func GetServerOverview(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	store, err := core.GetSSHProfileStore()
	if err != nil {
		log.Println("server overview store:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), overviewRequestTimeout)
	defer cancel()
	overview := core.CollectOverview(ctx, store)
	responseBody.Data = overview
	return &responseBody
}
