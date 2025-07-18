package sync

import (
	"fmt"
	"log"
	"time"

	"github.com/arimakouyou/pihole-sync/internal/config"
	"github.com/arimakouyou/pihole-sync/internal/pihole"
)

type Syncer struct {
	config      *config.Config
	masterClient *pihole.Client
	slaveClients []*pihole.Client
	lastSync    time.Time
}

type SyncResult struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	SyncedAt  time.Time `json:"synced_at"`
	Details   []SlaveResult `json:"details"`
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

	log.Println("Starting synchronization...")
	
	masterData, err := s.masterClient.GetData()
	if err != nil {
		return nil, fmt.Errorf("failed to get master data: %w", err)
	}

	var details []SlaveResult
	allSuccess := true

	for i, slaveClient := range s.slaveClients {
		slave := s.config.Slaves[i]
		result := s.syncSlave(slaveClient, slave, masterData)
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

func (s *Syncer) syncSlave(client *pihole.Client, slave config.SlaveConfig, masterData *pihole.PiholeData) SlaveResult {
	result := SlaveResult{
		Host:   slave.Host,
		Result: "ok",
	}

	filteredData := s.filterDataForSlave(masterData, slave.SyncItems)

	retryCount := 0
	maxRetries := s.config.SyncRetry.Count
	if !s.config.SyncRetry.Enabled {
		maxRetries = 0
	}

	for retryCount <= maxRetries {
		err := client.UpdateData(filteredData)
		if err == nil {
			break
		}

		if retryCount == maxRetries {
			result.Result = "error"
			result.Error = err.Error()
			log.Printf("Failed to sync slave %s after %d retries: %v", slave.Host, maxRetries, err)
			break
		}

		retryCount++
		log.Printf("Sync failed for slave %s, retrying (%d/%d): %v", slave.Host, retryCount, maxRetries, err)
		time.Sleep(time.Duration(retryCount) * time.Second)
	}

	return result
}

func (s *Syncer) filterDataForSlave(data *pihole.PiholeData, syncItems config.SyncItems) *pihole.PiholeData {
	filtered := &pihole.PiholeData{}

	if syncItems.Adlists {
		filtered.Adlists = data.Adlists
	}
	if syncItems.Blacklist {
		filtered.Blacklist = data.Blacklist
	}
	if syncItems.Whitelist {
		filtered.Whitelist = data.Whitelist
	}
	if syncItems.Groups {
		filtered.Groups = data.Groups
	}
	if syncItems.DNSRecords {
		filtered.DNSRecords = data.DNSRecords
	}
	if syncItems.DHCP {
		filtered.DHCP = data.DHCP
	}

	return filtered
}
