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
	"go.uber.org/zap"

	"github.com/arimakouyou/pihole-sync/internal/api"
	"github.com/arimakouyou/pihole-sync/internal/config"
	"github.com/arimakouyou/pihole-sync/internal/logger"
	"github.com/arimakouyou/pihole-sync/internal/metrics"
)

func main() {
	// まず基本的なログ出力で起動を通知
	log.Println("pihole-sync サーバー起動")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
		cfg = &config.Config{}
	}

	// Initialize zap logger based on configuration
	if err := logger.InitLogger(cfg); err != nil {
		log.Fatalf("ロガーの初期化に失敗しました: %v", err)
	}
	defer logger.Cleanup()

	// Use zap logger from now on
	logger.Logger.Info("pihole-sync server starting with zap logger")

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
			logger.Logger.Info("Running scheduled sync...")
			result, err := server.GetSyncer().Sync()
			if err != nil {
				logger.Logger.Error("Scheduled sync error", zap.Error(err))
				return
			}
			if result.Success {
				logger.Logger.Info("Scheduled sync completed successfully", zap.String("message", result.Message))
			} else {
				logger.Logger.Warn("Scheduled sync failed", zap.String("message", result.Message))
			}
		})
		if err != nil {
			logger.Logger.Error("Failed to add scheduled sync", zap.Error(err))
		} else {
			cronScheduler.Start()
			logger.Logger.Info("Started scheduled sync", zap.String("cron", cfg.SyncTrigger.Schedule))
		}
	}

	// Start Pi-hole file watcher if enabled
	var watcher *fsnotify.Watcher
	if cfg.SyncTrigger.PiholeFileWatch {
		var err error
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			logger.Logger.Error("Failed to create Pi-hole file watcher", zap.Error(err))
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
					logger.Logger.Warn("Failed to watch file", zap.String("file", file), zap.Error(err))
				} else {
					watchedFiles++
				}
			}

			if watchedFiles == 0 {
				logger.Logger.Warn("No Pi-hole files could be watched, closing watcher")
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
								logger.Logger.Debug("Pi-hole file changed", zap.String("file", event.Name))

								// Reset debounce timer
								if debounceTimer != nil {
									debounceTimer.Stop()
								}

								debounceTimer = time.AfterFunc(debounceDelay, func() {
									logger.Logger.Info("Debounce period completed, triggering sync after Pi-hole file changes")
									result, err := server.GetSyncer().Sync()
									if err != nil {
										logger.Logger.Error("Pi-hole file change sync error", zap.Error(err))
										return
									}
									if result.Success {
										logger.Logger.Info("Pi-hole file change sync completed", zap.String("message", result.Message))
									} else {
										logger.Logger.Warn("Pi-hole file change sync failed", zap.String("message", result.Message))
									}
								})
							}
						case err, ok := <-watcher.Errors:
							if !ok {
								return
							}
							logger.Logger.Error("Pi-hole file watcher error", zap.Error(err))
						case <-ctx.Done():
							if debounceTimer != nil {
								debounceTimer.Stop()
							}
							return
						}
					}
				}()
				logger.Logger.Info("Started Pi-hole file watch",
					zap.Int("watched_files", watchedFiles),
					zap.String("directory", piholeDataDir))
			}
		}
	}

	// Start metrics collection for all Pi-hole instances if metrics are enabled
	if cfg.Metrics.Enabled {
		metricsLogger := logger.Logger.With(zap.String("component", "metrics"))
		collector := metrics.NewCollector(cfg, metricsLogger)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := collector.Start(ctx); err != nil && err != context.Canceled {
				logger.Logger.Error("Metrics collector error", zap.Error(err))
			}
		}()

		instanceCount := 1 + len(cfg.Slaves) // master + slaves
		logger.Logger.Info("Started metrics collection", zap.Int("instances", instanceCount))
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
		logger.Logger.Info("Starting HTTP server", zap.Int("port", 8080))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Logger.Info("Shutdown signal received, gracefully shutting down")

	// Stop scheduled sync
	if cronScheduler != nil {
		logger.Logger.Info("Stopping scheduled sync")
		cronScheduler.Stop()
	}

	// Stop Pi-hole file watcher
	if watcher != nil {
		logger.Logger.Info("Stopping Pi-hole file watcher")
		watcher.Close()
	}

	// Cancel context to stop metrics collection
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Wait for all goroutines to finish
	wg.Wait()
	logger.Logger.Info("Server shutdown completed")
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
