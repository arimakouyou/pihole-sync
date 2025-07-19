package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/arimakouyou/pihole-sync/internal/api"
	"github.com/arimakouyou/pihole-sync/internal/config"
)

func main() {
	log.Println("pihole-sync サーバー起動")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
		cfg = &config.Config{}
	}

	server := api.NewServer(cfg)

	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	r.HandleFunc("/", server.IndexHandler)
	r.HandleFunc("/sync", server.SyncHandler)
	r.HandleFunc("/gravity", server.GravityGetHandler).Methods("GET")
	r.HandleFunc("/gravity", server.GravityPostHandler).Methods("POST")
	r.HandleFunc("/gravity/edit", server.GravityHandler)
	r.HandleFunc("/backup", server.BackupHandler)
	r.HandleFunc("/restore", server.RestoreHandler)
	r.HandleFunc("/config", server.ConfigHandler).Methods("GET")
	r.HandleFunc("/api/config", server.ConfigAPIHandler).Methods("GET")
	r.HandleFunc("/config", server.ConfigSaveHandler).Methods("POST")
	r.Handle("/metrics", promhttp.Handler())

	log.Println("サーバーをポート8080で起動中...")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
	}
}
