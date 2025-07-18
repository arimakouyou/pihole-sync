package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  api_key: "test-master-key"
slaves:
  - host: "http://test-slave.local"
    api_key: "test-slave-key"
    sync_items:
      adlists: true
      blacklist: false
      whitelist: true
      groups: false
      dns_records: true
      dhcp: false
logging:
  level: "DEBUG"
  debug: true
slack:
  webhook_url: "https://hooks.slack.com/test"
  notify_on_error: true
sync_retry:
  enabled: true
  count: 5
`

	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, "http://test-master.local", config.Master.Host)
	assert.Equal(t, "test-master-key", config.Master.APIKey)
	assert.Len(t, config.Slaves, 1)
	assert.Equal(t, "http://test-slave.local", config.Slaves[0].Host)
	assert.Equal(t, "test-slave-key", config.Slaves[0].APIKey)
	assert.True(t, config.Slaves[0].SyncItems.Adlists)
	assert.False(t, config.Slaves[0].SyncItems.Blacklist)
	assert.True(t, config.Slaves[0].SyncItems.Whitelist)
	assert.Equal(t, "DEBUG", config.Logging.Level)
	assert.True(t, config.Logging.Debug)
	assert.Equal(t, "https://hooks.slack.com/test", config.Slack.WebhookURL)
	assert.True(t, config.Slack.NotifyOnError)
	assert.True(t, config.SyncRetry.Enabled)
	assert.Equal(t, 5, config.SyncRetry.Count)
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent.yaml")
	assert.Error(t, err)
}

func TestSaveConfig(t *testing.T) {
	config := &Config{
		Master: MasterConfig{
			Host:   "http://test.local",
			APIKey: "test-key",
		},
		Logging: Logging{
			Level: "INFO",
			Debug: false,
		},
	}

	tmpFile, err := os.CreateTemp("", "config-save-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = config.SaveConfig(tmpFile.Name())
	require.NoError(t, err)

	loadedConfig, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, config.Master.Host, loadedConfig.Master.Host)
	assert.Equal(t, config.Master.APIKey, loadedConfig.Master.APIKey)
	assert.Equal(t, config.Logging.Level, loadedConfig.Logging.Level)
	assert.Equal(t, config.Logging.Debug, loadedConfig.Logging.Debug)
}

func TestSyncItemsAllFalse(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  api_key: "test-master-key"
slaves:
  - host: "http://test-slave.local"
    api_key: "test-slave-key"
    sync_items:
      adlists: false
      blacklist: false
      whitelist: false
      groups: false
      dns_records: false
      dhcp: false
`
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.False(t, config.Slaves[0].SyncItems.Adlists)
	assert.False(t, config.Slaves[0].SyncItems.Blacklist)
	assert.False(t, config.Slaves[0].SyncItems.Whitelist)
	assert.False(t, config.Slaves[0].SyncItems.Groups)
	assert.False(t, config.Slaves[0].SyncItems.DNSRecords)
	assert.False(t, config.Slaves[0].SyncItems.DHCP)
}

func TestSyncItemsAllTrue(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  api_key: "test-master-key"
slaves:
  - host: "http://test-slave.local"
    api_key: "test-slave-key"
    sync_items:
      adlists: true
      blacklist: true
      whitelist: true
      groups: true
      dns_records: true
      dhcp: true
`
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.True(t, config.Slaves[0].SyncItems.Adlists)
	assert.True(t, config.Slaves[0].SyncItems.Blacklist)
	assert.True(t, config.Slaves[0].SyncItems.Whitelist)
	assert.True(t, config.Slaves[0].SyncItems.Groups)
	assert.True(t, config.Slaves[0].SyncItems.DNSRecords)
	assert.True(t, config.Slaves[0].SyncItems.DHCP)
}

func TestSyncItemsMixed(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  api_key: "test-master-key"
slaves:
  - host: "http://test-slave.local"
    api_key: "test-slave-key"
    sync_items:
      adlists: true
      blacklist: false
      whitelist: true
      groups: false
      dns_records: true
      dhcp: false
`
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.True(t, config.Slaves[0].SyncItems.Adlists)
	assert.False(t, config.Slaves[0].SyncItems.Blacklist)
	assert.True(t, config.Slaves[0].SyncItems.Whitelist)
	assert.False(t, config.Slaves[0].SyncItems.Groups)
	assert.True(t, config.Slaves[0].SyncItems.DNSRecords)
	assert.False(t, config.Slaves[0].SyncItems.DHCP)
}

func TestSyncRetryNegativeCount(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  api_key: "test-master-key"
sync_retry:
  enabled: true
  count: -1
`
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, -1, config.SyncRetry.Count)
}

func TestSyncRetryExtremeValues(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{"zero", 0},
		{"large", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configData := fmt.Sprintf(`
master:
  host: "http://test-master.local"
  api_key: "test-master-key"
sync_retry:
  enabled: true
  count: %d
`, tt.count)
			tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(configData)
			require.NoError(t, err)
			tmpFile.Close()

			config, err := LoadConfig(tmpFile.Name())
			require.NoError(t, err)

			assert.Equal(t, tt.count, config.SyncRetry.Count)
		})
	}
}

func TestEmptyMasterFields(t *testing.T) {
	configData := `
master:
  host: ""
  api_key: ""
`
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Empty(t, config.Master.Host)
	assert.Empty(t, config.Master.APIKey)
}

func TestEmptySlaveFields(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  api_key: "test-master-key"
slaves:
  - host: ""
    api_key: ""
`
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Len(t, config.Slaves, 1)
	assert.Empty(t, config.Slaves[0].Host)
	assert.Empty(t, config.Slaves[0].APIKey)
}

func TestInvalidYAML(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  api_key: "test-master-key"
invalid_yaml: [
`
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	_, err = LoadConfig(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}
