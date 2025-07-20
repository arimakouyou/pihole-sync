package sync

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/arimakouyou/pihole-sync/internal/config"
	"github.com/arimakouyou/pihole-sync/internal/logger"
	"github.com/arimakouyou/pihole-sync/internal/pihole"
)

type Syncer struct {
	config       *config.Config
	masterClient *pihole.Client
	slaveClients []*pihole.Client
	lastSync     time.Time
}

type SyncResult struct {
	Success  bool          `json:"success"`
	Message  string        `json:"message"`
	SyncedAt time.Time     `json:"synced_at"`
	Details  []SlaveResult `json:"details"`
}

type SlaveResult struct {
	Host   string `json:"host"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

func NewSyncer(cfg *config.Config) *Syncer {
	masterClient := pihole.NewClient(cfg.Master.Host, cfg.Master.Password)

	var slaveClients []*pihole.Client
	for _, slave := range cfg.Slaves {
		slaveClients = append(slaveClients, pihole.NewClient(slave.Host, slave.Password))
	}

	return &Syncer{
		config:       cfg,
		masterClient: masterClient,
		slaveClients: slaveClients,
	}
}

func (s *Syncer) CanSync() bool {
	return time.Since(s.lastSync) >= 10*time.Second
}

func (s *Syncer) GetLastSync() time.Time {
	return s.lastSync
}

func (s *Syncer) Sync() (*SyncResult, error) {
	if !s.CanSync() {
		return &SyncResult{
			Success: false,
			Message: "10秒以内に呼び出し済みのため、処理は実行されませんでした",
		}, nil
	}

	// Safe logging - check if logger is initialized
	if logger.Logger != nil {
		logger.Logger.Info("Starting synchronization")
	}

	// Teleporter APIを使用してマスターからバックアップをダウンロード
	masterBackup, err := s.masterClient.GetBackup()
	if err != nil {
		return nil, fmt.Errorf("failed to get master backup: %w", err)
	}

	var details []SlaveResult
	allSuccess := true

	for i, slaveClient := range s.slaveClients {
		slave := s.config.Slaves[i]
		result := s.syncSlaveWithBackup(slaveClient, slave, masterBackup)
		details = append(details, result)

		if result.Result != "ok" {
			allSuccess = false
		}
	}

	s.lastSync = time.Now()

	syncResult := &SyncResult{
		Success:  allSuccess,
		SyncedAt: s.lastSync,
		Details:  details,
	}

	if allSuccess {
		syncResult.Message = "同期完了"
	} else {
		syncResult.Message = "同期中にエラーが発生しました"
	}

	return syncResult, nil
}

func (s *Syncer) syncSlaveWithBackup(client *pihole.Client, slave config.SlaveConfig, masterBackup []byte) SlaveResult {
	result := SlaveResult{
		Host:   slave.Host,
		Result: "ok",
	}

	// 設定に基づいてインポートオプションを生成
	importOptions := s.generateImportOptions(slave.SyncItems)

	retryCount := 0
	maxRetries := s.config.SyncRetry.Count
	if !s.config.SyncRetry.Enabled {
		maxRetries = 0
	}

	for retryCount <= maxRetries {
		err := client.RestoreBackupWithOptions(masterBackup, importOptions)
		if err == nil {
			if logger.Logger != nil {
				logger.Logger.Info("Successfully synced slave using Teleporter API",
					zap.String("host", slave.Host))
			}
			break
		}

		if retryCount == maxRetries {
			result.Result = "error"
			result.Error = err.Error()
			if logger.Logger != nil {
				logger.Logger.Error("Failed to sync slave after retries",
					zap.String("host", slave.Host),
					zap.Int("max_retries", maxRetries),
					zap.Error(err))
			}
			break
		}

		retryCount++
		if logger.Logger != nil {
			logger.Logger.Warn("Sync failed for slave, retrying",
				zap.String("host", slave.Host),
				zap.Int("retry", retryCount),
				zap.Int("max_retries", maxRetries),
				zap.Error(err))
		}
		time.Sleep(time.Duration(retryCount) * time.Second)
	}

	return result
}

// generateImportOptions converts SyncItems config to Pi-hole import options
func (s *Syncer) generateImportOptions(syncItems config.SyncItems) map[string]bool {
	return map[string]bool{
		"adlists":     syncItems.Adlists,
		"blacklist":   syncItems.Blacklist,
		"whitelist":   syncItems.Whitelist,
		"regex":       syncItems.Regex,
		"groups":      syncItems.Groups,
		"dns_records": syncItems.DNSRecords,
		"dhcp":        syncItems.DHCP,
		"clients":     syncItems.Clients,
		"settings":    syncItems.Settings,
	}
}
