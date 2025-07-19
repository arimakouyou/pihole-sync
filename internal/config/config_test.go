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
  password: "test-master-password"
slaves:
  - host: "http://test-slave.local"
    password: "test-slave-password"
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
	assert.Equal(t, "test-master-password", config.Master.Password)
	assert.Len(t, config.Slaves, 1)
	assert.Equal(t, "http://test-slave.local", config.Slaves[0].Host)
	assert.Equal(t, "test-slave-password", config.Slaves[0].Password)
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
			Host:     "http://test.local",
			Password: "test-password",
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
	assert.Equal(t, config.Master.Password, loadedConfig.Master.Password)
	assert.Equal(t, config.Logging.Level, loadedConfig.Logging.Level)
	assert.Equal(t, config.Logging.Debug, loadedConfig.Logging.Debug)
}

func TestSyncItemsAllFalse(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  password: "test-master-password"
slaves:
  - host: "http://test-slave.local"
    password: "test-slave-password"
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
  password: "test-master-password"
slaves:
  - host: "http://test-slave.local"
    password: "test-slave-password"
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
  password: "test-master-password"
slaves:
  - host: "http://test-slave.local"
    password: "test-slave-password"
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
  password: "test-master-password"
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
  password: "test-master-password"
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
  password: ""
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
	assert.Empty(t, config.Master.Password)
}

func TestEmptySlaveFields(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  password: "test-master-password"
slaves:
  - host: ""
    password: ""
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
	assert.Empty(t, config.Slaves[0].Password)
}

func TestInvalidYAML(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  password: "test-master-password"
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

func TestSaveConfigWithComplexData(t *testing.T) {
	config := &Config{
		Master: MasterConfig{
			Host:     "http://complex-master.local",
			Password: "complex-password-123!@#",
		},
		Slaves: []SlaveConfig{
			{
				Host:     "http://slave1.local",
				Password: "slave1-password",
				SyncItems: SyncItems{
					Adlists:    true,
					Blacklist:  false,
					Whitelist:  true,
					Groups:     false,
					DNSRecords: true,
					DHCP:       false,
				},
			},
			{
				Host:     "http://slave2.local",
				Password: "slave2-password",
				SyncItems: SyncItems{
					Adlists:    false,
					Blacklist:  true,
					Whitelist:  false,
					Groups:     true,
					DNSRecords: false,
					DHCP:       true,
				},
			},
		},
		SyncTrigger: SyncTrigger{
			Schedule:        "0 */6 * * *",
			APICall:         true,
			WebUI:           false,
			PiholeFileWatch: true,
		},
		Logging: Logging{
			Level: "DEBUG",
			Debug: true,
		},
		Slack: Slack{
			WebhookURL:    "https://hooks.slack.com/services/TEST/TEST/TEST",
			NotifyOnError: true,
		},
		SyncRetry: SyncRetry{
			Enabled: true,
			Count:   10,
		},
	}

	tmpFile, err := os.CreateTemp("", "config-complex-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = config.SaveConfig(tmpFile.Name())
	require.NoError(t, err)

	loadedConfig, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, config.Master.Host, loadedConfig.Master.Host)
	assert.Equal(t, config.Master.Password, loadedConfig.Master.Password)
	assert.Len(t, loadedConfig.Slaves, 2)
	assert.Equal(t, config.Slaves[0].Host, loadedConfig.Slaves[0].Host)
	assert.Equal(t, config.Slaves[0].SyncItems.Adlists, loadedConfig.Slaves[0].SyncItems.Adlists)
	assert.Equal(t, config.SyncTrigger.Schedule, loadedConfig.SyncTrigger.Schedule)
	assert.Equal(t, config.Logging.Level, loadedConfig.Logging.Level)
	assert.Equal(t, config.Slack.WebhookURL, loadedConfig.Slack.WebhookURL)
	assert.Equal(t, config.SyncRetry.Count, loadedConfig.SyncRetry.Count)
}

func TestLoadConfigWithMissingOptionalFields(t *testing.T) {
	configData := `
master:
  host: "http://test-master.local"
  password: "test-password"
slaves:
  - host: "http://test-slave.local"
    password: "test-password"
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
	assert.Equal(t, "test-password", config.Master.Password)
	assert.Len(t, config.Slaves, 1)
	assert.Equal(t, "http://test-slave.local", config.Slaves[0].Host)
	
	assert.False(t, config.Slaves[0].SyncItems.Adlists)
	assert.False(t, config.Slaves[0].SyncItems.Blacklist)
	assert.Empty(t, config.SyncTrigger.Schedule)
	assert.Empty(t, config.Logging.Level)
	assert.Empty(t, config.Slack.WebhookURL)
	assert.False(t, config.SyncRetry.Enabled)
}

func TestSaveConfigFilePermissions(t *testing.T) {
	config := &Config{
		Master: MasterConfig{
			Host:     "http://test.local",
			Password: "test-password",
		},
	}

	tmpFile, err := os.CreateTemp("", "config-permissions-test-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = config.SaveConfig(tmpFile.Name())
	require.NoError(t, err)

	fileInfo, err := os.Stat(tmpFile.Name())
	require.NoError(t, err)
	
	actualPerms := fileInfo.Mode().Perm()
	assert.True(t, actualPerms&0644 == 0644 || actualPerms == 0600, 
		"File permissions should be 0644 or 0600, got %o", actualPerms)
}
