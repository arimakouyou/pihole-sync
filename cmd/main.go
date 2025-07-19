package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"

	"github.com/arimakouyou/pihole-sync/internal/api"
	"github.com/arimakouyou/pihole-sync/internal/config"
	"github.com/arimakouyou/pihole-sync/internal/metrics"
)

func main() {
	log.Println("pihole-sync サーバー起動")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
		cfg = &config.Config{}
	}

	// Set default values for metrics configuration if not specified
	setDefaultMetricsConfig(&cfg.Metrics)

	server := api.NewServer(cfg)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start scheduled sync if enabled
	var cronScheduler *cron.Cron
	if cfg.SyncTrigger.Schedule != "" {
		cronScheduler = cron.New()
		_, err := cronScheduler.AddFunc(cfg.SyncTrigger.Schedule, func() {
			log.Println("Running scheduled sync...")
			result, err := server.GetSyncer().Sync()
			if err != nil {
				log.Printf("Scheduled sync error: %v", err)
				return
			}
			if result.Success {
				log.Printf("Scheduled sync completed successfully: %s", result.Message)
			} else {
				log.Printf("Scheduled sync failed: %s", result.Message)
			}
		})
		if err != nil {
			log.Printf("Failed to add scheduled sync: %v", err)
		} else {
			cronScheduler.Start()
			log.Printf("Started scheduled sync with cron expression: %s", cfg.SyncTrigger.Schedule)
		}
	}

	// Start Pi-hole file watcher if enabled
	var watcher *fsnotify.Watcher
	if cfg.SyncTrigger.PiholeFileWatch {
		var err error
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			log.Printf("Failed to create Pi-hole file watcher: %v", err)
		} else {
			// Pi-hole data directory (mounted from host /etc/pihole)
			piholeDataDir := "/var/lib/pihole"
			watchFiles := []string{
				piholeDataDir + "/dhcp.leases",
				piholeDataDir + "/gravity.db",
				piholeDataDir + "/pihole.toml",
			}

			watchedFiles := 0
			for _, file := range watchFiles {
				err = watcher.Add(file)
				if err != nil {
					log.Printf("Failed to watch %s: %v", file, err)
				} else {
					watchedFiles++
				}
			}

			if watchedFiles == 0 {
				log.Println("No Pi-hole files could be watched, closing watcher")
				watcher.Close()
				watcher = nil
			} else {
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer watcher.Close()

					var debounceTimer *time.Timer
					const debounceDelay = 10 * time.Second

					for {
						select {
						case event, ok := <-watcher.Events:
							if !ok {
								return
							}
							if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
								log.Printf("Pi-hole file changed: %s", event.Name)

								// Reset debounce timer
								if debounceTimer != nil {
									debounceTimer.Stop()
								}

								debounceTimer = time.AfterFunc(debounceDelay, func() {
									log.Printf("Debounce period completed, triggering sync after Pi-hole file changes...")
									result, err := server.GetSyncer().Sync()
									if err != nil {
										log.Printf("Pi-hole file change sync error: %v", err)
										return
									}
									if result.Success {
										log.Printf("Pi-hole file change sync completed: %s", result.Message)
									} else {
										log.Printf("Pi-hole file change sync failed: %s", result.Message)
									}
								})
							}
						case err, ok := <-watcher.Errors:
							if !ok {
								return
							}
							log.Printf("Pi-hole file watcher error: %v", err)
						case <-ctx.Done():
							if debounceTimer != nil {
								debounceTimer.Stop()
							}
							return
						}
					}
				}()
				log.Printf("Started Pi-hole file watch for %d files in %s", watchedFiles, piholeDataDir)
			}
		}
	}

	// Start metrics collection for all Pi-hole instances if metrics are enabled
	if cfg.Metrics.Enabled {
		logger := log.New(os.Stdout, "[METRICS] ", log.LstdFlags)
		collector := metrics.NewCollector(cfg, logger)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := collector.Start(ctx); err != nil && err != context.Canceled {
				log.Printf("Metrics collector error: %v", err)
			}
		}()

		instanceCount := 1 + len(cfg.Slaves) // master + slaves
		log.Printf("Started metrics collection for %d Pi-hole instances", instanceCount)
	}

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

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start HTTP server in a separate goroutine
	go func() {
		log.Println("サーバーをポート8080で起動中...")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("サーバーの起動に失敗しました: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, gracefully shutting down...")

	// Stop scheduled sync
	if cronScheduler != nil {
		log.Println("Stopping scheduled sync...")
		cronScheduler.Stop()
	}

	// Stop Pi-hole file watcher
	if watcher != nil {
		log.Println("Stopping Pi-hole file watcher...")
		watcher.Close()
	}

	// Cancel context to stop metrics collection
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("Server shutdown completed")
}

// setDefaultMetricsConfig sets default values for metrics configuration
func setDefaultMetricsConfig(cfg *config.MetricsConfig) {
	if cfg.CollectionInterval == 0 {
		cfg.CollectionInterval = 30 * time.Second
	}
	if cfg.TopItemsLimit == 0 {
		cfg.TopItemsLimit = 10
	}
}
