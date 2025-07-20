package metrics

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/arimakouyou/pihole-sync/internal/config"
	"github.com/arimakouyou/pihole-sync/internal/pihole"
)

// PiholeInstance represents a Pi-hole instance with its client and metadata
type PiholeInstance struct {
	Client *pihole.Client
	Host   string
	Role   string // "master" or "slave"
}

// Collector is responsible for collecting Pi-hole metrics from multiple instances
type Collector struct {
	instances []PiholeInstance
	config    *config.MetricsConfig
	logger    *zap.Logger
}

// NewCollector creates a new metrics collector for multiple Pi-hole instances
func NewCollector(cfg *config.Config, logger *zap.Logger) *Collector {
	var instances []PiholeInstance

	// Add master instance
	masterClient := pihole.NewClient(cfg.Master.Host, cfg.Master.Password)
	instances = append(instances, PiholeInstance{
		Client: masterClient,
		Host:   cfg.Master.Host,
		Role:   "master",
	})

	// Add slave instances
	for _, slave := range cfg.Slaves {
		slaveClient := pihole.NewClient(slave.Host, slave.Password)
		instances = append(instances, PiholeInstance{
			Client: slaveClient,
			Host:   slave.Host,
			Role:   "slave",
		})
	}

	return &Collector{
		instances: instances,
		config:    &cfg.Metrics,
		logger:    logger,
	}
}

// Start begins the metrics collection process
func (c *Collector) Start(ctx context.Context) error {
	if !c.config.Enabled {
		c.logger.Info("Metrics collection is disabled")
		return nil
	}

	c.logger.Info("Starting Pi-hole metrics collector", zap.Duration("interval", c.config.CollectionInterval))

	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	// Collect metrics immediately on start
	c.collectMetrics()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping metrics collector")
			return ctx.Err()
		case <-ticker.C:
			c.collectMetrics()
		}
	}
}

// collectMetrics collects all enabled metrics from all Pi-hole instances
func (c *Collector) collectMetrics() {
	startTime := time.Now()
	c.logger.Debug("Starting metrics collection", zap.Int("instances", len(c.instances)))

	for _, instance := range c.instances {
		c.collectInstanceMetrics(instance)
	}

	duration := time.Since(startTime)
	c.logger.Debug("Metrics collection completed", zap.Duration("duration", duration))
}

// collectInstanceMetrics collects metrics from a single Pi-hole instance
func (c *Collector) collectInstanceMetrics(instance PiholeInstance) {
	instanceName := instance.Host
	role := instance.Role

	// Collect summary statistics
	if err := c.collectSummaryStats(instance); err != nil {
		c.logger.Error("Failed to collect summary stats", 
			zap.String("host", instanceName), 
			zap.String("role", role), 
			zap.Error(err))
		RecordAPIError(instanceName, role, "stats/summary")
	}

	// Collect query types
	if err := c.collectQueryTypes(instance); err != nil {
		c.logger.Error("Failed to collect query types", 
			zap.String("host", instanceName), 
			zap.String("role", role), 
			zap.Error(err))
		RecordAPIError(instanceName, role, "stats/query_types")
	}

	// Collect upstreams if enabled
	if c.config.EnableUpstreams {
		if err := c.collectUpstreams(instance); err != nil {
			c.logger.Error("Failed to collect upstreams", 
				zap.String("host", instanceName), 
				zap.String("role", role), 
				zap.Error(err))
			RecordAPIError(instanceName, role, "stats/upstreams")
		}
	}

	// Collect top domains if enabled
	if c.config.EnableTopDomains {
		if err := c.collectTopDomains(instance); err != nil {
			c.logger.Error("Failed to collect top domains", 
				zap.String("host", instanceName), 
				zap.String("role", role), 
				zap.Error(err))
			RecordAPIError(instanceName, role, "stats/top_domains")
		}
	}

	// Collect top clients if enabled
	if c.config.EnableTopClients {
		if err := c.collectTopClients(instance); err != nil {
			c.logger.Error("Failed to collect top clients", 
				zap.String("host", instanceName), 
				zap.String("role", role), 
				zap.Error(err))
			RecordAPIError(instanceName, role, "stats/top_clients")
		}
	}

	// Record successful collection
	RecordSuccessfulCollection(instanceName, role)
}

// collectSummaryStats collects and updates summary statistics for an instance
func (c *Collector) collectSummaryStats(instance PiholeInstance) error {
	startTime := time.Now()
	stats, err := instance.Client.GetSummaryStats()
	if err != nil {
		return err
	}

	UpdateSummaryStats(stats, instance.Host, instance.Role)
	RecordAPIResponseTime(instance.Host, instance.Role, "stats/summary", time.Since(startTime).Seconds())
	return nil
}

// collectQueryTypes collects and updates query type statistics for an instance
func (c *Collector) collectQueryTypes(instance PiholeInstance) error {
	startTime := time.Now()
	queryTypes, err := instance.Client.GetQueryTypes()
	if err != nil {
		return err
	}

	UpdateQueryTypes(queryTypes, instance.Host, instance.Role)
	RecordAPIResponseTime(instance.Host, instance.Role, "stats/query_types", time.Since(startTime).Seconds())
	return nil
}

// collectUpstreams collects and updates upstream server statistics for an instance
func (c *Collector) collectUpstreams(instance PiholeInstance) error {
	startTime := time.Now()
	upstreams, err := instance.Client.GetUpstreams()
	if err != nil {
		return err
	}

	UpdateUpstreams(upstreams, instance.Host, instance.Role)
	RecordAPIResponseTime(instance.Host, instance.Role, "stats/upstreams", time.Since(startTime).Seconds())
	return nil
}

// collectTopDomains collects and updates top domains statistics for an instance
func (c *Collector) collectTopDomains(instance PiholeInstance) error {
	startTime := time.Now()
	topDomains, err := instance.Client.GetTopDomains()
	if err != nil {
		return err
	}

	UpdateTopDomains(topDomains, instance.Host, instance.Role, c.config.TopItemsLimit)
	RecordAPIResponseTime(instance.Host, instance.Role, "stats/top_domains", time.Since(startTime).Seconds())
	return nil
}

// collectTopClients collects and updates top clients statistics for an instance
func (c *Collector) collectTopClients(instance PiholeInstance) error {
	startTime := time.Now()
	topClients, err := instance.Client.GetTopClients()
	if err != nil {
		return err
	}

	UpdateTopClients(topClients, instance.Host, instance.Role, c.config.TopItemsLimit)
	RecordAPIResponseTime(instance.Host, instance.Role, "stats/top_clients", time.Since(startTime).Seconds())
	return nil
}
