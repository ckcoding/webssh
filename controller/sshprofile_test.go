package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

type savedSSHResponse struct {
	Msg  string                     `json:"Msg"`
	Data map[string]json.RawMessage `json:"Data"`
}

type savedSSHItem struct {
	ID          string `json:"id"`
	Host        string `json:"host"`
	IP          string `json:"ip"`
	CountryFlag string `json:"countryFlag"`
	Location    string `json:"location"`
	OS          string `json:"os"`
	Memory      string `json:"memory"`
	ExpiresAt   string `json:"expiresAt"`
	Username    string `json:"username"`
	Port        int    `json:"port"`
	Password    string `json:"password"`
	LoginType   int    `json:"logintype"`
}

func TestSavedSSHEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("WEBSSH_PROFILE_FILE", filepath.Join(t.TempDir(), "ssh_profiles.json"))

	router := gin.New()
	router.GET("/ssh/saved", func(c *gin.Context) {
		c.JSON(http.StatusOK, ListSavedSSH(c))
	})
	router.POST("/ssh/saved", func(c *gin.Context) {
		c.JSON(http.StatusOK, SaveSavedSSH(c))
	})
	router.DELETE("/ssh/saved/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, DeleteSavedSSH(c))
	})

	initial := performSavedSSHRequest(t, router, http.MethodGet, "/ssh/saved", nil)
	if initial.Msg != "success" {
		t.Fatalf("expected initial list success, got %q", initial.Msg)
	}
	if list := decodeSavedSSHList(t, initial); len(list) != 0 {
		t.Fatalf("expected empty initial list, got %d items", len(list))
	}

	createPayload := map[string]interface{}{
		"host":        "192.168.1.2",
		"ip":          "203.0.113.10",
		"countryFlag": "🇸🇬",
		"location":    "Singapore",
		"os":          "Ubuntu 24.04",
		"memory":      "8GB",
		"expiresAt":   "2027-01-01 00:00:00",
		"username":    "root",
		"port":        22,
		"password":    "secret",
		"logintype":   0,
	}
	created := performSavedSSHRequest(t, router, http.MethodPost, "/ssh/saved", createPayload)
	if created.Msg != "success" {
		t.Fatalf("expected create success, got %q", created.Msg)
	}
	createdList := decodeSavedSSHList(t, created)
	if len(createdList) != 1 {
		t.Fatalf("expected 1 saved profile after create, got %d", len(createdList))
	}
	if createdList[0].Password != "secret" {
		t.Fatalf("expected saved password to be returned")
	}
	if createdList[0].IP != "203.0.113.10" || createdList[0].CountryFlag != "🇸🇬" || createdList[0].Location != "Singapore" || createdList[0].OS != "Ubuntu 24.04" || createdList[0].Memory != "8GB" || createdList[0].ExpiresAt != "2027-01-01 00:00:00" {
		t.Fatalf("expected metadata to be returned, got %+v", createdList[0])
	}

	updatePayload := map[string]interface{}{
		"host":        "192.168.1.2",
		"ip":          "203.0.113.20",
		"countryFlag": "🇯🇵",
		"location":    "Tokyo",
		"os":          "Debian 12",
		"memory":      "16GB",
		"expiresAt":   "",
		"username":    "root",
		"port":        22,
		"password":    "changed",
		"logintype":   0,
	}
	updated := performSavedSSHRequest(t, router, http.MethodPost, "/ssh/saved", updatePayload)
	if updated.Msg != "success" {
		t.Fatalf("expected update success, got %q", updated.Msg)
	}
	updatedList := decodeSavedSSHList(t, updated)
	if len(updatedList) != 1 {
		t.Fatalf("expected update to keep 1 saved profile, got %d", len(updatedList))
	}
	if updatedList[0].ID != createdList[0].ID {
		t.Fatalf("expected update to reuse profile id")
	}
	if updatedList[0].Password != "changed" {
		t.Fatalf("expected updated password to be returned")
	}
	if updatedList[0].IP != "203.0.113.20" || updatedList[0].CountryFlag != "🇯🇵" || updatedList[0].Location != "Tokyo" || updatedList[0].OS != "Debian 12" || updatedList[0].Memory != "16GB" || updatedList[0].ExpiresAt != "" {
		t.Fatalf("expected updated metadata to be returned, got %+v", updatedList[0])
	}

	deleted := performSavedSSHRequest(t, router, http.MethodDelete, "/ssh/saved/"+createdList[0].ID, nil)
	if deleted.Msg != "success" {
		t.Fatalf("expected delete success, got %q", deleted.Msg)
	}
	if list := decodeSavedSSHList(t, deleted); len(list) != 0 {
		t.Fatalf("expected empty list after delete, got %d items", len(list))
	}
}

func performSavedSSHRequest(t *testing.T, router *gin.Engine, method, path string, payload interface{}) savedSSHResponse {
	t.Helper()

	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		content, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		body = bytes.NewReader(content)
	}

	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var response savedSSHResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, body=%s", err, recorder.Body.String())
	}
	return response
}

func decodeSavedSSHList(t *testing.T, response savedSSHResponse) []savedSSHItem {
	t.Helper()

	rawList, ok := response.Data["list"]
	if !ok {
		t.Fatalf("expected response data to contain list")
	}

	var list []savedSSHItem
	if err := json.Unmarshal(rawList, &list); err != nil {
		t.Fatalf("json.Unmarshal(list) error = %v", err)
	}
	return list
}
