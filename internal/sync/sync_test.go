package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/arimakouyou/pihole-sync/internal/config"
	"github.com/arimakouyou/pihole-sync/internal/pihole"
)

func TestNewSyncer(t *testing.T) {
	cfg := &config.Config{
		Master: config.MasterConfig{
			Host:   "http://master.local",
			APIKey: "master-key",
		},
		Slaves: []config.SlaveConfig{
			{
				Host:   "http://slave.local",
				APIKey: "slave-key",
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

func TestFilterDataForSlave(t *testing.T) {
	cfg := &config.Config{}
	syncer := NewSyncer(cfg)

	masterData := &pihole.PiholeData{
		Adlists:    []string{"adlist1", "adlist2"},
		Blacklist:  []string{"black1", "black2"},
		Whitelist:  []string{"white1", "white2"},
		Groups:     []string{"group1", "group2"},
		DNSRecords: []string{"dns1", "dns2"},
		DHCP:       []string{"dhcp1", "dhcp2"},
	}

	syncItems := config.SyncItems{
		Adlists:    true,
		Blacklist:  false,
		Whitelist:  true,
		Groups:     false,
		DNSRecords: true,
		DHCP:       false,
	}

	filtered := syncer.filterDataForSlave(masterData, syncItems)

	assert.Equal(t, []string{"adlist1", "adlist2"}, filtered.Adlists)
	assert.Empty(t, filtered.Blacklist)
	assert.Equal(t, []string{"white1", "white2"}, filtered.Whitelist)
	assert.Empty(t, filtered.Groups)
	assert.Equal(t, []string{"dns1", "dns2"}, filtered.DNSRecords)
	assert.Empty(t, filtered.DHCP)
}
