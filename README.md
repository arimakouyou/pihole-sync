# Pi-hole Sync

Pi-hole同期システム - 複数台のPi-holeの設定を同期するための管理・同期システム

## 概要

このシステムは、マスター/スレーブ構成で複数台のPi-holeの設定を同期します。API・WebUI・定期実行・ファイル変更検知など多様な同期トリガーを持ち、設定・gravityリストの管理、Slack通知、監視、バックアップ/復元機能を提供します。

## 機能

- **Pi-hole設定の同期**: API経由、WebUI、定期、設定ファイル変更検知
- **gravityリストの管理**: 取得・編集・同期
- **マスター/スレーブ構成**: 同期対象項目はSlaveごとに選択可能
- **バックアップ/復元**: 設定・gravityリストのJSON形式での保存・復元
- **Slack通知**: エラー時の通知機能
- **ログ出力**: 標準出力、ログレベル制御
- **同期リトライ**: 回数設定可能
- **メトリクス監視**: Prometheus対応
- **WebUI**: 設定編集、gravity編集、バックアップ/復元画面

## セットアップ

### 1. 設定ファイルの作成

`config.yaml`ファイルを作成し、マスター・スレーブのPi-hole情報を設定してください：

```yaml
master:
  host: "http://pihole-master.local"
  password: "your-master-application-password"
slaves:
  - host: "http://pihole-slave1.local"
    password: "your-slave1-application-password"
    sync_items:
      adlists: true
      blacklist: true
      whitelist: false
      groups: true
      dns_records: false
      dhcp: false
sync_trigger:
  schedule: "0 3 * * *"
  api_call: true
  webui: true
  config_file_watch: true
logging:
  level: "INFO"
  debug: false
slack:
  webhook_url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
  notify_on_error: true
sync_retry:
  enabled: true
  count: 3
```

### 2. ビルドと実行

```bash
# 依存関係のインストール
go mod tidy

# ビルド
go build -o pihole-sync ./cmd/main.go

# 実行
./pihole-sync
```

### 3. Dockerでの実行

```bash
# Dockerイメージのビルド
docker build -t pihole-sync .

# 実行
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml pihole-sync
```

## API エンドポイント

### POST /sync
即時同期を実行します（rate limit: 10秒）

```bash
curl -X POST http://localhost:8080/sync
```

### GET /gravity
gravityリストを取得します

```bash
# JSON形式
curl -H "Accept: application/json" http://localhost:8080/gravity

# テキスト形式
curl http://localhost:8080/gravity
```

### POST /gravity
gravityリストを更新します

```bash
# JSON形式
curl -X POST -H "Content-Type: application/json" \
  -d '{"gravity": ["ads.example.com", "tracker.example.com"]}' \
  http://localhost:8080/gravity

# テキスト形式
curl -X POST -H "Content-Type: text/plain" \
  -d "address=/ads.example.com/0.0.0.0
address=/tracker.example.com/0.0.0.0" \
  http://localhost:8080/gravity
```

### GET /backup
設定・gravityリストのJSONファイルをダウンロードします

```bash
curl http://localhost:8080/backup -o backup.json
```

### POST /restore
JSONファイルから設定・gravityリストを復元します

```bash
curl -X POST -H "Content-Type: application/json" \
  -d @backup.json http://localhost:8080/restore
```

### GET /metrics
Prometheus形式のメトリクスを取得します

```bash
curl http://localhost:8080/metrics
```

## WebUI

ブラウザで `http://localhost:8080` にアクセスすると、管理画面が表示されます。

- **トップページ**: 同期実行、各機能へのナビゲーション
- **設定編集**: YAML設定ファイルの編集
- **Gravity編集**: gravityリストの編集
- **バックアップ/復元**: ファイルのダウンロード・アップロード

## 開発

### テストの実行

```bash
go test ./...
```

### 静的解析

```bash
staticcheck ./...
```

### ビルドスクリプト

```bash
./build.sh
```

## ライセンス

MIT License
