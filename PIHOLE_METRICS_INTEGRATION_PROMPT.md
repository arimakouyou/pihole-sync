# Pi-hole Metrics Integration Enhancement Prompt

## 現在の状況

### 既存のメトリクス実装
`pihole-sync`アプリケーションは現在、以下の基本的なPrometheusメトリクスを収集・公開しています：

**既存メトリクス（`internal/metrics/metrics.go`）:**
- `pihole_sync_success_total` - 同期成功回数
- `pihole_sync_failure_total` - 同期失敗回数
- `pihole_gravity_edit_total` - Gravityリスト編集回数
- `pihole_api_call_total` - API呼び出し回数
- `pihole_error_total` - エラー発生回数

これらのメトリクスは`http://localhost:8080/metrics`エンドポイントでPrometheus形式で公開されています。

### 目標
Pi-hole FTL APIから取得できる詳細なメトリクスを統合し、包括的な監視ダッシュボードを構築できるようにする。

## タスク: Pi-hole FTL APIメトリクス統合の実装

### 1. 新しいメトリクス収集器の実装

`internal/metrics/pihole_metrics.go`を新規作成し、以下のPi-hole FTL APIエンドポイントからメトリクスを収集する実装を行ってください：

#### Core Statistics (`/stats/summary`)
```go
// 統計サマリーメトリクス
pihole_domains_blocked_total           // ブロックされたドメイン数
pihole_dns_queries_today_total         // 今日のDNSクエリ数
pihole_ads_blocked_today_total         // 今日のブロック数
pihole_ads_percentage_today            // 今日のブロック率
pihole_unique_domains_total            // ユニークドメイン数
pihole_queries_forwarded_total         // 転送されたクエリ数
pihole_queries_cached_total            // キャッシュされたクエリ数
pihole_clients_ever_seen_total         // 全クライアント数
pihole_unique_clients_total            // ユニーククライアント数
pihole_dns_queries_all_types_total     // 全タイプのDNSクエリ数
pihole_reply_unknown_total             // 不明な応答数
pihole_reply_nodata_total              // NODATAレスポンス数
pihole_reply_nxdomain_total            // NXDOMAINレスポンス数
pihole_reply_cname_total               // CNAMEレスポンス数
pihole_reply_ip_total                  // IPアドレスレスポンス数
pihole_privacy_level                   // プライバシーレベル設定
pihole_status_enabled                  // Pi-hole有効状態（1=有効, 0=無効）
```

#### Query Types (`/stats/query_types`)
```go
// クエリタイプ別メトリクス（ラベル付き）
pihole_query_types_total{type="A"}        // A レコードクエリ数
pihole_query_types_total{type="AAAA"}     // AAAA レコードクエリ数
pihole_query_types_total{type="ANY"}      // ANY レコードクエリ数
pihole_query_types_total{type="SRV"}      // SRV レコードクエリ数
pihole_query_types_total{type="SOA"}      // SOA レコードクエリ数
pihole_query_types_total{type="PTR"}      // PTR レコードクエリ数
pihole_query_types_total{type="TXT"}      // TXT レコードクエリ数
pihole_query_types_total{type="NAPTR"}    // NAPTR レコードクエリ数
pihole_query_types_total{type="MX"}       // MX レコードクエリ数
pihole_query_types_total{type="DS"}       // DS レコードクエリ数
pihole_query_types_total{type="RRSIG"}    // RRSIG レコードクエリ数
pihole_query_types_total{type="DNSKEY"}   // DNSKEY レコードクエリ数
pihole_query_types_total{type="NS"}       // NS レコードクエリ数
pihole_query_types_total{type="OTHER"}    // その他のクエリ数
```

#### Upstream Servers (`/stats/upstreams`)
```go
// アップストリームサーバー別メトリクス（ラベル付き）
pihole_upstream_queries_total{upstream="8.8.8.8"}     // アップストリーム別クエリ数
pihole_upstream_response_time_seconds{upstream="8.8.8.8"} // アップストリーム別応答時間
```

#### Top Domains and Clients (`/stats/top_domains`, `/stats/top_clients`)
```go
// トップドメイン/クライアントメトリクス（ラベル付き）
pihole_top_permitted_domains_total{domain="example.com"}    // 許可ドメイン別クエリ数
pihole_top_blocked_domains_total{domain="ads.example.com"}  // ブロックドメイン別クエリ数
pihole_top_clients_total{client="192.168.1.100"}           // クライアント別クエリ数
```

#### DNS Cache Metrics (FTL specific)
```go
// DNSキャッシュメトリクス
pihole_dns_cache_size                  // キャッシュサイズ
pihole_dns_cache_insertions_total      // キャッシュ挿入数
pihole_dns_cache_evictions_total       // キャッシュエビクション数
pihole_dns_queries_forwarded_total     // 転送クエリ数
pihole_dns_queries_answered_locally_total  // ローカル応答数
```
### 2. API クライアントの拡張

`internal/pihole/client.go`を拡張し、以下のメソッドを追加：

```go
// 統計情報取得メソッド
func (c *Client) GetSummaryStats() (*SummaryStats, error)
func (c *Client) GetQueryTypes() (*QueryTypes, error)
func (c *Client) GetUpstreams() (*Upstreams, error)
func (c *Client) GetTopDomains() (*TopDomains, error)
func (c *Client) GetTopClients() (*TopClients, error)
func (c *Client) GetRecentBlocked() (*RecentBlocked, error)
```

### 3. 設定拡張

`internal/config/config.go`に以下の設定項目を追加：

```go
type Config struct {
    // 既存フィールド...
    
    Metrics MetricsConfig `yaml:"metrics"`
}

type MetricsConfig struct {
    Enabled              bool          `yaml:"enabled" default:"true"`
    CollectionInterval   time.Duration `yaml:"collection_interval" default:"30s"`
    EnableTopDomains     bool          `yaml:"enable_top_domains" default:"true"`
    EnableTopClients     bool          `yaml:"enable_top_clients" default:"true"`
    EnableUpstreams      bool          `yaml:"enable_upstreams" default:"true"`
    EnableCacheMetrics   bool          `yaml:"enable_cache_metrics" default:"true"`
    TopItemsLimit        int           `yaml:"top_items_limit" default:"10"`
}
```

### 4. バックグラウンド収集タスク

`internal/metrics/collector.go`を新規作成し、定期的にPi-hole APIからメトリクスを収集するゴルーチンを実装：

```go
type Collector struct {
    piholeClient *pihole.Client
    config       *config.MetricsConfig
    logger       *log.Logger
}

func (c *Collector) Start(ctx context.Context) error {
    ticker := time.NewTicker(c.config.CollectionInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            c.collectMetrics()
        }
    }
}

func (c *Collector) collectMetrics() {
    // 各エンドポイントからデータを取得してPrometheusメトリクスを更新
}
```

### 5. エラーハンドリングと監視

以下のエラー監視メトリクスも追加：

```go
pihole_api_errors_total{endpoint="/stats/summary"}     // エンドポイント別APIエラー数
pihole_api_response_time_seconds{endpoint="/stats/summary"} // エンドポイント別応答時間
pihole_last_successful_collection_timestamp            // 最後の成功した収集時刻
```

### 6. 実装における技術的考慮事項

#### Rate Limiting
- Pi-hole APIの負荷を考慮し、適切な間隔（デフォルト30秒）で収集
- 設定可能な収集間隔を提供

#### Error Resilience
- APIエラー時にメトリクス収集を停止せず、エラーをメトリクスとして記録
- 指数バックオフによる再試行機能

#### Memory Management
- 大量のラベル（トップドメイン/クライアント）によるメモリ使用量の管理
- 設定可能な上限値

#### Performance
- ゴルーチンによる非同期収集
- 既存の同期処理に影響しない設計

### 7. テスト戦略

以下のテストを含む`*_test.go`ファイルを作成：

```go
// ユニットテスト
func TestPiholeMetricsCollection(t *testing.T)
func TestAPIErrorHandling(t *testing.T)
func TestMetricsRegistration(t *testing.T)

// 統合テスト
func TestEndToEndMetricsCollection(t *testing.T)
```

### 8. 設定例

`config.yaml`に以下のサンプル設定を追加：

```yaml
metrics:
  enabled: true
  collection_interval: "30s"
  enable_top_domains: true
  enable_top_clients: true
  enable_upstreams: true
  enable_cache_metrics: true
  top_items_limit: 10
```

### 9. ドキュメント

実装後、以下のドキュメントを更新：

1. `README.md` - 新機能の説明
2. `docs/metrics.md` - 利用可能なメトリクスの詳細リスト
3. `docs/prometheus-grafana.md` - Grafanaダッシュボード設定例

### 10. 期待される結果

実装完了後、以下が実現されます：

- `/metrics`エンドポイントでPi-holeの詳細統計が取得可能
- Grafanaでの包括的なPi-hole監視ダッシュボード構築
- DNS性能とブロック効果の可視化
- アラート設定によるプロアクティブな監視

### 11. サンプル Grafanaダッシュボード用クエリ

実装後に使用できるPromQLクエリ例：

```promql
# 今日のブロック率
pihole_ads_percentage_today

# 1時間あたりのクエリ数
rate(pihole_dns_queries_today_total[1h])

# トップブロックドメイン
topk(10, pihole_top_blocked_domains_total)

# アップストリーム応答時間
pihole_upstream_response_time_seconds

# クエリタイプ分布
sum by (type) (pihole_query_types_total)
```

このプロンプトに基づいて実装を行うことで、現在の基本的なメトリクスから、Pi-holeの包括的な監視が可能な高度なメトリクス収集システムへと拡張できます。
