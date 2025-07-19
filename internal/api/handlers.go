package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/arimakouyou/pihole-sync/internal/config"
	"github.com/arimakouyou/pihole-sync/internal/metrics"
	"github.com/arimakouyou/pihole-sync/internal/notifications"
	"github.com/arimakouyou/pihole-sync/internal/sync"
)

type Server struct {
	config   *config.Config
	syncer   *sync.Syncer
	notifier *notifications.SlackNotifier
	gravity  []string
}

type BackupData struct {
	Config  *config.Config `json:"config"`
	Gravity []string       `json:"gravity"`
}

func NewServer(cfg *config.Config) *Server {
	syncer := sync.NewSyncer(cfg)
	notifier := notifications.NewSlackNotifier(cfg.Slack.WebhookURL, cfg.Slack.NotifyOnError)

	return &Server{
		config:   cfg,
		syncer:   syncer,
		notifier: notifier,
		gravity:  cfg.Gravity,
	}
}

func (s *Server) SyncHandler(w http.ResponseWriter, r *http.Request) {
	metrics.IncrementAPICall()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.syncer.CanSync() {
		response := map[string]interface{}{
			"status":         "error",
			"message":        "10秒以内に呼び出し済みのため、処理は実行されませんでした",
			"last_synced_at": time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	result, err := s.syncer.Sync()
	if err != nil {
		metrics.IncrementError()
		s.notifier.NotifyError("同期エラー", err.Error())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		response := map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	if result.Success {
		metrics.IncrementSyncSuccess()
	} else {
		metrics.IncrementSyncFailure()
		s.notifier.NotifyError("同期失敗", result.Message)
	}

	response := map[string]interface{}{
		"status":    "success",
		"message":   result.Message,
		"synced_at": result.SyncedAt.Format(time.RFC3339),
		"details": map[string]interface{}{
			"slaves": result.Details,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) GravityGetHandler(w http.ResponseWriter, r *http.Request) {
	metrics.IncrementAPICall()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/json") {
		response := map[string]interface{}{
			"gravity": s.gravity,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		for _, entry := range s.gravity {
			fmt.Fprintf(w, "0.0.0.0 %s\n", entry)
		}
	}
}

func (s *Server) GravityPostHandler(w http.ResponseWriter, r *http.Request) {
	metrics.IncrementAPICall()
	metrics.IncrementGravityEdit()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		metrics.IncrementError()
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var request map[string][]string
		if err := json.Unmarshal(body, &request); err != nil {
			metrics.IncrementError()
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if gravity, ok := request["gravity"]; ok {
			s.gravity = gravity
			s.config.Gravity = gravity

			if err := s.config.SaveConfig("config.yaml"); err != nil {
				log.Printf("Failed to save config with gravity: %v", err)
			}
		}
	} else {
		text := string(body)
		lines := strings.Split(text, "\n")
		s.gravity = []string{}

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			var domain string
			if strings.HasPrefix(line, "address=/") && strings.HasSuffix(line, "/0.0.0.0") {
				domain = strings.TrimPrefix(line, "address=/")
				domain = strings.TrimSuffix(domain, "/0.0.0.0")
			} else if strings.HasPrefix(line, "0.0.0.0 ") {
				domain = strings.TrimPrefix(line, "0.0.0.0 ")
			} else {
				domain = line
			}

			if domain != "" {
				s.gravity = append(s.gravity, domain)
			}
		}

		s.config.Gravity = s.gravity
		if err := s.config.SaveConfig("config.yaml"); err != nil {
			log.Printf("Failed to save config with gravity: %v", err)
		}
	}

	response := map[string]string{
		"status":  "success",
		"message": "gravityリストを更新しました",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) BackupHandler(w http.ResponseWriter, r *http.Request) {
	metrics.IncrementAPICall()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	backup := BackupData{
		Config:  s.config,
		Gravity: s.gravity,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=pihole-sync-backup.json")

	if err := json.NewEncoder(w).Encode(backup); err != nil {
		metrics.IncrementError()
		log.Printf("Failed to encode backup data: %v", err)
		http.Error(w, "Failed to create backup", http.StatusInternalServerError)
	}
}

func (s *Server) RestoreHandler(w http.ResponseWriter, r *http.Request) {
	metrics.IncrementAPICall()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		metrics.IncrementError()
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var backup BackupData
	if err := json.Unmarshal(body, &backup); err != nil {
		metrics.IncrementError()
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.config = backup.Config
	s.gravity = backup.Gravity
	s.config.Gravity = backup.Gravity

	if err := s.config.SaveConfig("config.yaml"); err != nil {
		log.Printf("Failed to save config: %v", err)
	}

	response := map[string]string{
		"status":  "success",
		"message": "設定・gravityリストを復元しました",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/index.html")
}

func (s *Server) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/config.html")
}

func (s *Server) ConfigAPIHandler(w http.ResponseWriter, r *http.Request) {
	configData, err := os.ReadFile("config.yaml")
	if err != nil {
		// ファイルが存在しない場合は空の文字列を返す
		configData = []byte("")
	}
	response := map[string]string{"config": string(configData)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) ConfigSaveHandler(w http.ResponseWriter, r *http.Request) {
	var request map[string]string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	configData, ok := request["config"]
	if !ok {
		http.Error(w, "Missing config data", http.StatusBadRequest)
		return
	}

	if err := os.WriteFile("config.yaml", []byte(configData), 0644); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	response := map[string]string{"status": "success", "message": "設定を保存しました"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) GravityHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/gravity.html")
}
