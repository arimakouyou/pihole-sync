# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o pihole-sync ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/pihole-sync ./pihole-sync
COPY --from=builder /app/web ./web
COPY --from=builder /app/config.yaml ./config.yaml
CMD ["./pihole-sync"]

