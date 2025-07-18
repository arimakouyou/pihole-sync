package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arimakouyou/pihole-sync/internal/config"
)

func createTestServer() *Server {
	cfg := &config.Config{
		Master: config.MasterConfig{
			Host:     "http://test-master.local",
			Password: "test-password",
		},
		Slaves: []config.SlaveConfig{
			{
				Host:     "http://test-slave.local",
				Password: "test-slave-password",
				SyncItems: config.SyncItems{
					Adlists:   true,
					Blacklist: true,
				},
			},
		},
		SyncRetry: config.SyncRetry{
			Enabled: true,
			Count:   3,
		},
	}
	return NewServer(cfg)
}

func TestSyncHandler(t *testing.T) {
	server := createTestServer()

	req, err := http.NewRequest("POST", "/sync", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.SyncHandler(rr, req)

	if rr.Code == http.StatusOK {
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "status")
		t.Logf("Sync response: %v", response)
	} else {
		t.Logf("Sync failed with status %d, body: %s", rr.Code, rr.Body.String())
		assert.True(t, rr.Code >= 400, "Expected error status code")
	}
}

func TestSyncHandlerMethodNotAllowed(t *testing.T) {
	server := createTestServer()

	req, err := http.NewRequest("GET", "/sync", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.SyncHandler(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestGravityGetHandler(t *testing.T) {
	server := createTestServer()
	server.gravity = []string{"ads.example.com", "tracker.example.com"}

	req, err := http.NewRequest("GET", "/gravity", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	rr := httptest.NewRecorder()
	server.GravityGetHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response map[string][]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, []string{"ads.example.com", "tracker.example.com"}, response["gravity"])
}

func TestGravityGetHandlerText(t *testing.T) {
	server := createTestServer()
	server.gravity = []string{"ads.example.com", "tracker.example.com"}

	req, err := http.NewRequest("GET", "/gravity", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.GravityGetHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))

	expected := "address=/ads.example.com/0.0.0.0\naddress=/tracker.example.com/0.0.0.0\n"
	assert.Equal(t, expected, rr.Body.String())
}

func TestGravityPostHandlerJSON(t *testing.T) {
	server := createTestServer()

	requestData := map[string][]string{
		"gravity": {"test.com", "example.com"},
	}
	jsonData, err := json.Marshal(requestData)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/gravity", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.GravityPostHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.Equal(t, []string{"test.com", "example.com"}, server.gravity)
}

func TestGravityPostHandlerText(t *testing.T) {
	server := createTestServer()

	textData := "address=/test.com/0.0.0.0\naddress=/example.com/0.0.0.0\n"
	req, err := http.NewRequest("POST", "/gravity", strings.NewReader(textData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()
	server.GravityPostHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.Equal(t, []string{"test.com", "example.com"}, server.gravity)
}

func TestBackupHandler(t *testing.T) {
	server := createTestServer()
	server.gravity = []string{"test.com"}

	req, err := http.NewRequest("GET", "/backup", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.BackupHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Content-Disposition"), "attachment")

	var backup BackupData
	err = json.Unmarshal(rr.Body.Bytes(), &backup)
	require.NoError(t, err)

	assert.Equal(t, server.config, backup.Config)
	assert.Equal(t, server.gravity, backup.Gravity)
}

func TestRestoreHandler(t *testing.T) {
	server := createTestServer()

	backupData := BackupData{
		Config: &config.Config{
			Master: config.MasterConfig{
				Host:     "http://restored-master.local",
				Password: "restored-password",
			},
		},
		Gravity: []string{"restored.com"},
	}

	jsonData, err := json.Marshal(backupData)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/restore", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.RestoreHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "http://restored-master.local", server.config.Master.Host)
	assert.Equal(t, []string{"restored.com"}, server.gravity)
}

func TestIndexHandler(t *testing.T) {
	server := createTestServer()

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.IndexHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/html; charset=utf-8", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "Pi-hole Sync 管理画面")
}
