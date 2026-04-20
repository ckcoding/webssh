package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCheckSSHConnectionRejectsEmptyHost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/ssh/check", func(c *gin.Context) {
		c.JSON(http.StatusOK, CheckSSHConnection(c))
	})

	content, err := json.Marshal(map[string]interface{}{
		"host":      "",
		"username":  "root",
		"port":      22,
		"password":  "secret",
		"logintype": 0,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/ssh/check", bytes.NewReader(content))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var response ResponseBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, body=%s", err, recorder.Body.String())
	}
	if response.Msg != "host is required" {
		t.Fatalf("expected host validation error, got %q", response.Msg)
	}
}
