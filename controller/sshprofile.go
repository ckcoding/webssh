package controller

import (
	"context"
	"log"
	"strings"
	"time"
	"webssh/core"

	"github.com/gin-gonic/gin"
)

// ListSavedSSH 获取已保存 SSH 配置
func ListSavedSSH(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}

	store, err := core.GetSSHProfileStore()
	if err != nil {
		log.Println("get ssh profile store:", err)
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

	var profile core.SavedSSHProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		log.Println("save ssh profile bind:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	store, err := core.GetSSHProfileStore()
	if err != nil {
		log.Println("get ssh profile store:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	if duplicate, exists, err := store.FindDuplicate(profile); err != nil {
		log.Println("find duplicate ssh profile:", err)
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
		log.Println("save ssh profile:", err)
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

	id := strings.TrimSpace(c.Param("id"))
	store, err := core.GetSSHProfileStore()
	if err != nil {
		log.Println("get ssh profile store:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	if err := store.Delete(id); err != nil {
		log.Println("delete ssh profile:", err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	responseBody.Data = map[string]interface{}{
		"list": store.List(),
	}
	return &responseBody
}

// RefreshSavedSSHInfo collects system metadata using server-side saved credentials.
func RefreshSavedSSHInfo(c *gin.Context) *ResponseBody {
	response := ResponseBody{Msg: "success"}
	store, err := core.GetSSHProfileStore()
	if err != nil {
		response.Msg = err.Error()
		return &response
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	profile, info, err := core.RefreshSavedSSHMetadata(ctx, store, c.Param("id"))
	if err != nil {
		response.Msg = err.Error()
		return &response
	}
	response.Data = map[string]interface{}{"profile": profile, "info": info}
	return &response
}
