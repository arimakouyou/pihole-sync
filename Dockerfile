# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder

# セキュリティパッケージの更新とgitのインストール
RUN apk update && apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Go modulesのキャッシュ効率化
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピーしてビルド
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o pihole-sync ./cmd/main.go

# 最終イメージ
FROM scratch

# CA証明書とタイムゾーン情報をコピー
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# バイナリとWebファイルをコピー
COPY --from=builder /app/pihole-sync /pihole-sync
COPY --from=builder /app/web /web

# 非rootユーザーとして実行
USER 65534:65534

EXPOSE 8080

CMD ["/pihole-sync"]
