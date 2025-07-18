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
		gravity:  []string{},
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
			"status": "error",
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
			fmt.Fprintf(w, "address=/%s/0.0.0.0\n", entry)
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
		}
	} else {
		text := string(body)
		lines := strings.Split(text, "\n")
		s.gravity = []string{}
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && strings.HasPrefix(line, "address=/") && strings.HasSuffix(line, "/0.0.0.0") {
				domain := strings.TrimPrefix(line, "address=/")
				domain = strings.TrimSuffix(domain, "/0.0.0.0")
				s.gravity = append(s.gravity, domain)
			}
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
	html := `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Pi-hole Sync</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .button-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .btn { display: block; padding: 15px 20px; background-color: #007cba; color: white; text-decoration: none; border-radius: 5px; text-align: center; font-weight: bold; transition: background-color 0.3s; }
        .btn:hover { background-color: #005a87; }
        .btn.danger { background-color: #dc3545; }
        .btn.danger:hover { background-color: #c82333; }
        .status { padding: 15px; background-color: #e9ecef; border-radius: 5px; margin-top: 20px; }
        textarea { width: 100%; height: 300px; margin: 10px 0; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-family: monospace; }
        input[type="file"] { margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Pi-hole Sync 管理画面</h1>
        
        <div class="button-grid">
            <button class="btn" onclick="performSync()">同期実行</button>
            <a href="/config" class="btn">設定編集</a>
            <a href="/gravity/edit" class="btn">Gravity編集</a>
            <a href="/backup" class="btn">バックアップ</a>
            <button class="btn" onclick="showRestore()">復元</button>
        </div>

        <div id="restore-section" style="display: none;">
            <h3>復元</h3>
            <input type="file" id="restore-file" accept=".json">
            <button class="btn" onclick="performRestore()">復元実行</button>
        </div>

        <div class="status" id="status">
            ステータス: 待機中
        </div>
    </div>

    <script>
        function performSync() {
            document.getElementById('status').innerHTML = 'ステータス: 同期中...';
            fetch('/sync', { method: 'POST' })
                .then(response => response.json())
                .then(data => {
                    document.getElementById('status').innerHTML = 'ステータス: ' + data.message;
                })
                .catch(error => {
                    document.getElementById('status').innerHTML = 'ステータス: エラー - ' + error.message;
                });
        }

        function showRestore() {
            document.getElementById('restore-section').style.display = 'block';
        }

        function performRestore() {
            const fileInput = document.getElementById('restore-file');
            const file = fileInput.files[0];
            if (!file) {
                alert('ファイルを選択してください');
                return;
            }

            const reader = new FileReader();
            reader.onload = function(e) {
                fetch('/restore', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: e.target.result
                })
                .then(response => response.json())
                .then(data => {
                    document.getElementById('status').innerHTML = 'ステータス: ' + data.message;
                    document.getElementById('restore-section').style.display = 'none';
                })
                .catch(error => {
                    document.getElementById('status').innerHTML = 'ステータス: エラー - ' + error.message;
                });
            };
            reader.readAsText(file);
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (s *Server) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		configData, err := os.ReadFile("config.yaml")
		if err != nil {
			configData = []byte("")
		}

		html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>設定編集 - Pi-hole Sync</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 1000px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-bottom: 30px; }
        textarea { width: 100%%; height: 400px; margin: 10px 0; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-family: monospace; }
        .btn { padding: 10px 20px; background-color: #007cba; color: white; border: none; border-radius: 5px; cursor: pointer; margin-right: 10px; }
        .btn:hover { background-color: #005a87; }
        .btn.secondary { background-color: #6c757d; }
        .btn.secondary:hover { background-color: #545b62; }
    </style>
</head>
<body>
    <div class="container">
        <h1>設定編集</h1>
        <form method="post">
            <textarea name="config" placeholder="YAML設定を入力してください...">%s</textarea>
            <br>
            <button type="submit" class="btn">保存</button>
            <a href="/" class="btn secondary">戻る</a>
        </form>
    </div>
</body>
</html>`, string(configData))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	} else if r.Method == http.MethodPost {
		configData := r.FormValue("config")
		if err := os.WriteFile("config.yaml", []byte(configData), 0644); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (s *Server) GravityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		gravityText := ""
		for _, entry := range s.gravity {
			gravityText += fmt.Sprintf("address=/%s/0.0.0.0\n", entry)
		}

		html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gravity編集 - Pi-hole Sync</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 1000px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-bottom: 30px; }
        textarea { width: 100%%; height: 400px; margin: 10px 0; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-family: monospace; }
        .btn { padding: 10px 20px; background-color: #007cba; color: white; border: none; border-radius: 5px; cursor: pointer; margin-right: 10px; }
        .btn:hover { background-color: #005a87; }
        .btn.secondary { background-color: #6c757d; }
        .btn.secondary:hover { background-color: #545b62; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Gravity編集</h1>
        <form method="post">
            <textarea name="gravity" placeholder="Gravityリストを入力してください...">%s</textarea>
            <br>
            <button type="submit" class="btn">保存</button>
            <a href="/" class="btn secondary">戻る</a>
        </form>
    </div>
</body>
</html>`, gravityText)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	} else if r.Method == http.MethodPost {
		gravityData := r.FormValue("gravity")
		
		r.Body = io.NopCloser(strings.NewReader(gravityData))
		r.Header.Set("Content-Type", "text/plain")
		
		s.GravityPostHandler(w, r)
		if w.Header().Get("Location") == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}
}
