package controller

import (
	"fmt"
	"time"
	"webssh/core"

	"github.com/gin-gonic/gin"
)

// GetSSHSystemInfo 获取远端服务器系统信息
func GetSSHSystemInfo(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)

	sshInfo := c.DefaultQuery("sshInfo", "")
	sshClient, err := core.DecodedMsgToSSHClient(sshInfo)
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}

	if err := sshClient.GenerateClient(); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	defer sshClient.Close()

	info, err := core.CollectSSHSystemInfo(&sshClient)
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
