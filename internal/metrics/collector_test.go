package metrics

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/arimakouyou/pihole-sync/internal/config"
)

func TestNewCollector(t *testing.T) {
	cfg := &config.Config{
		Master: config.MasterConfig{
			Host:     "http://master.localhost",
			Password: "master-password",
		},
		Slaves: []config.SlaveConfig{
			{
				Host:     "http://slave1.localhost",
				Password: "slave1-password",
			},
			{
				Host:     "http://slave2.localhost",
				Password: "slave2-password",
			},
		},
		Metrics: config.MetricsConfig{
			Enabled:            true,
			CollectionInterval: 30 * time.Second,
			TopItemsLimit:      10,
		},
	}
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	collector := NewCollector(cfg, logger)

	if collector == nil {
		t.Fatal("Expected collector to be created, got nil")
	}

	if len(collector.instances) != 3 { // 1 master + 2 slaves
		t.Errorf("Expected 3 instances, got %d", len(collector.instances))
	}

	// Check master instance
	if collector.instances[0].Role != "master" {
		t.Errorf("Expected first instance to be master, got %s", collector.instances[0].Role)
	}
	if collector.instances[0].Host != "http://master.localhost" {
		t.Errorf("Expected master host to be http://master.localhost, got %s", collector.instances[0].Host)
	}

	// Check slave instances
	if collector.instances[1].Role != "slave" {
		t.Errorf("Expected second instance to be slave, got %s", collector.instances[1].Role)
	}
	if collector.instances[2].Role != "slave" {
		t.Errorf("Expected third instance to be slave, got %s", collector.instances[2].Role)
	}

	if collector.config != &cfg.Metrics {
		t.Error("Expected config to be set correctly")
	}

	if collector.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
}

func TestCollectorStart_Disabled(t *testing.T) {
	cfg := &config.Config{
		Master: config.MasterConfig{
			Host:     "http://localhost",
			Password: "password",
		},
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
	}
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	collector := NewCollector(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := collector.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error when metrics are disabled, got: %v", err)
	}
}

func TestCollectorStart_WithContext(t *testing.T) {
	cfg := &config.Config{
		Master: config.MasterConfig{
			Host:     "http://localhost",
			Password: "password",
		},
		Metrics: config.MetricsConfig{
			Enabled:            true,
			CollectionInterval: 100 * time.Millisecond, // Short interval for testing
			TopItemsLimit:      5,
		},
	}
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	collector := NewCollector(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := collector.Start(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got: %v", err)
	}
}
