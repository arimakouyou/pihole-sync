package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Master      MasterConfig  `yaml:"master"`
	Slaves      []SlaveConfig `yaml:"slaves"`
	SyncTrigger SyncTrigger   `yaml:"sync_trigger"`
	Logging     Logging       `yaml:"logging"`
	Slack       Slack         `yaml:"slack"`
	SyncRetry   SyncRetry     `yaml:"sync_retry"`
	Gravity     []string      `yaml:"gravity"`
	Metrics     MetricsConfig `yaml:"metrics"`
}

type MasterConfig struct {
	Host     string `yaml:"host"`
	Password string `yaml:"password"`
}

type SlaveConfig struct {
	Host      string    `yaml:"host"`
	Password  string    `yaml:"password"`
	SyncItems SyncItems `yaml:"sync_items"`
}

type SyncItems struct {
	Adlists    bool `yaml:"adlists"`
	Blacklist  bool `yaml:"blacklist"`
	Whitelist  bool `yaml:"whitelist"`
	Regex      bool `yaml:"regex"`
	Groups     bool `yaml:"groups"`
	DNSRecords bool `yaml:"dns_records"`
	DHCP       bool `yaml:"dhcp"`
	Clients    bool `yaml:"clients"`
	Settings   bool `yaml:"settings"`
}

type SyncTrigger struct {
	Schedule        string `yaml:"schedule"`
	APICall         bool   `yaml:"api_call"`
	WebUI           bool   `yaml:"webui"`
	ConfigFileWatch bool   `yaml:"config_file_watch"`
}

type Logging struct {
	Level string `yaml:"level"`
	Debug bool   `yaml:"debug"`
}

type Slack struct {
	WebhookURL    string `yaml:"webhook_url"`
	NotifyOnError bool   `yaml:"notify_on_error"`
}

type SyncRetry struct {
	Enabled bool `yaml:"enabled"`
	Count   int  `yaml:"count"`
}

type MetricsConfig struct {
	Enabled            bool          `yaml:"enabled"`
	CollectionInterval time.Duration `yaml:"collection_interval"`
	EnableTopDomains   bool          `yaml:"enable_top_domains"`
	EnableTopClients   bool          `yaml:"enable_top_clients"`
	EnableUpstreams    bool          `yaml:"enable_upstreams"`
	EnableCacheMetrics bool          `yaml:"enable_cache_metrics"`
	TopItemsLimit      int           `yaml:"top_items_limit"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func (c *Config) SaveConfig(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
