package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/arimakouyou/pihole-sync/internal/config"
)

func TestNewSyncer(t *testing.T) {
	cfg := &config.Config{
		Master: config.MasterConfig{
			Host:     "http://master.local",
			Password: "master-password",
		},
		Slaves: []config.SlaveConfig{
			{
				Host:     "http://slave.local",
				Password: "slave-password",
			},
		},
	}

	syncer := NewSyncer(cfg)

	assert.NotNil(t, syncer)
	assert.Equal(t, cfg, syncer.config)
	assert.NotNil(t, syncer.masterClient)
	assert.Len(t, syncer.slaveClients, 1)
}

func TestCanSync(t *testing.T) {
	cfg := &config.Config{}
	syncer := NewSyncer(cfg)

	assert.True(t, syncer.CanSync())

	syncer.lastSync = time.Now()
	assert.False(t, syncer.CanSync())

	syncer.lastSync = time.Now().Add(-11 * time.Second)
	assert.True(t, syncer.CanSync())
}

func TestSyncWithRetryLogic(t *testing.T) {
	tests := []struct {
		name         string
		retryEnabled bool
		retryCount   int
		expectError  bool
	}{
		{"no retry", false, 0, true},
		{"retry once", true, 1, true},
		{"retry multiple", true, 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Master: config.MasterConfig{
					Host:     "http://invalid-master.local",
					Password: "test-password",
				},
				Slaves: []config.SlaveConfig{
					{
						Host:     "http://invalid-slave.local",
						Password: "test-password",
						SyncItems: config.SyncItems{
							Adlists: true,
						},
					},
				},
				SyncRetry: config.SyncRetry{
					Enabled: tt.retryEnabled,
					Count:   tt.retryCount,
				},
			}

			syncer := NewSyncer(cfg)
			result, err := syncer.Sync()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, result.Success)
			}
		})
	}
}

func TestSyncWithNilMasterData(t *testing.T) {
	cfg := &config.Config{
		Master: config.MasterConfig{
			Host:     "http://invalid-master.local",
			Password: "test-password",
		},
	}

	syncer := NewSyncer(cfg)
	_, err := syncer.Sync()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get master data")
}

func TestSyncWithMultipleSlaves(t *testing.T) {
	cfg := &config.Config{
		Master: config.MasterConfig{
			Host:     "http://invalid-master.local",
			Password: "test-password",
		},
		Slaves: []config.SlaveConfig{
			{
				Host:     "http://invalid-slave1.local",
				Password: "test-password1",
				SyncItems: config.SyncItems{
					Adlists:   true,
					Blacklist: false,
				},
			},
			{
				Host:     "http://invalid-slave2.local",
				Password: "test-password2",
				SyncItems: config.SyncItems{
					Adlists:   false,
					Blacklist: true,
				},
			},
		},
		SyncRetry: config.SyncRetry{
			Enabled: false,
			Count:   0,
		},
	}

	syncer := NewSyncer(cfg)
	_, err := syncer.Sync()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get master data")
}

func TestSyncRateLimiting(t *testing.T) {
	cfg := &config.Config{
		Master: config.MasterConfig{
			Host:     "http://test-master.local",
			Password: "test-password",
		},
	}

	syncer := NewSyncer(cfg)

	syncer.lastSync = time.Now().Add(-5 * time.Second)
	assert.False(t, syncer.CanSync(), "Should not allow sync within 10 seconds")

	result, err := syncer.Sync()
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "10秒以内に呼び出し済み")
}

func TestSyncWithDisabledRetry(t *testing.T) {
	cfg := &config.Config{
		Master: config.MasterConfig{
			Host:     "http://invalid-master.local",
			Password: "test-password",
		},
		Slaves: []config.SlaveConfig{
			{
				Host:     "http://invalid-slave.local",
				Password: "test-password",
				SyncItems: config.SyncItems{
					Adlists: true,
				},
			},
		},
		SyncRetry: config.SyncRetry{
			Enabled: false,
			Count:   5,
		},
	}

	syncer := NewSyncer(cfg)
	_, err := syncer.Sync()
	assert.Error(t, err)
}

func TestGetLastSyncInitialValue(t *testing.T) {
	cfg := &config.Config{}
	syncer := NewSyncer(cfg)

	lastSync := syncer.GetLastSync()
	assert.True(t, lastSync.IsZero(), "Initial last sync should be zero time")
}
