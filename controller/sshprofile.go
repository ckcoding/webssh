package controller

import (
	"fmt"
	"strings"
	"time"
	"webssh/core"

	"github.com/gin-gonic/gin"
)

// ListSavedSSH 获取已保存 SSH 配置
func ListSavedSSH(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	store, err := core.GetSSHProfileStore()
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	responseBody.Data = map[string]interface{}{
		"list": store.List(),
	}
	return &responseBody
}

// SaveSavedSSH 保存 SSH 配置
func SaveSavedSSH(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	var profile core.SavedSSHProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	store, err := core.GetSSHProfileStore()
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	if duplicate, exists, err := store.FindDuplicate(profile); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	} else if exists {
		responseBody.Msg = core.ErrSavedSSHAlreadyExists.Error()
		responseBody.Data = map[string]interface{}{
			"duplicate": true,
			"profile":   duplicate,
			"list":      store.List(),
		}
		return &responseBody
	}
	saved, err := store.Save(profile)
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	responseBody.Data = map[string]interface{}{
		"profile": saved,
		"list":    store.List(),
	}
	return &responseBody
}

// DeleteSavedSSH 删除 SSH 配置
func DeleteSavedSSH(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	id := strings.TrimSpace(c.Param("id"))
	store, err := core.GetSSHProfileStore()
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	if err := store.Delete(id); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	responseBody.Data = map[string]interface{}{
		"list": store.List(),
	}
	return &responseBody
}
