# syntax=docker/dockerfile:1

# ========================================
# Multi-service Dockerfile for Grapery
# Builds: server, vippay
# ========================================

FROM golang:1.25 AS builder
WORKDIR /src
COPY grapery/go.mod grapery/go.sum ./
RUN go mod download
COPY grapery/ ./

# Build services
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/grapery-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/grapery-vippay ./cmd/vippay

# ========================================
# Main API Server Image
# ========================================
FROM gcr.io/distroless/base-debian12 AS server
WORKDIR /app
COPY --from=builder /bin/grapery-server /app/grapery

# ========== Server Configuration ==========
ENV GRAPERY_ENV=production
ENV GRAPERY_HTTP_PORT=8080
ENV GRAPERY_LOG_LEVEL=info
ENV GRAPERY_ALLOW_ORIGIN=*

# ========== Database Configuration ==========
ENV DB_DATABASE=grapery
ENV DB_USERNAME=
ENV DB_PASSWORD=
ENV DB_ADDRESS=

# ========== Redis Configuration ==========
ENV REDIS_ADDRESS=
ENV REDIS_PASSWORD=
ENV REDIS_DATABASE=0
ENV REDIS_PING_INTERVAL=30

# ========== AI Configuration ==========
ENV HUOSHAN_API_KEY=
ENV HUOSHAN_BASE_URL=
ENV GEMINI_API_KEY=
ENV GEMINI_BASE_URL=
ENV AI_DEFAULT_PROVIDER=huoshan

# ========== JWT Configuration ==========
ENV JWT_SECRET=
ENV JWT_EXPIRY_HOURS=24

# ========== Aliyun OSS Configuration ==========
ENV ALIYUN_API_KEY=
ENV ALIYUN_SECRET_KEY=
ENV ALIYUN_ENDPOINT=oss-cn-shanghai.aliyuncs.com
ENV ALIYUN_BUCKET=grapery-dev
ENV ALIYUN_ROLE_ARN=

EXPOSE 8080
CMD ["/app/grapery"]

# ========================================
# VIP Payment Service Image
# ========================================
FROM gcr.io/distroless/base-debian12 AS vippay
WORKDIR /app
COPY --from=builder /bin/grapery-vippay /app/grapery-vippay

# ========== Server Configuration ==========
ENV VIPPAY_PORT=8081
ENV VIPPAY_DOMAIN=https://www.grapery.xyz
ENV LOG_LEVEL=info

# ========== Database Configuration ==========
ENV DB_DATABASE=grapery
ENV DB_USERNAME=
ENV DB_PASSWORD=
ENV DB_ADDRESS=

# ========== JWT Configuration ==========
ENV JWT_SECRET=
ENV JWT_EXPIRY_HOURS=24

# ========== Apple IAP Configuration ==========
ENV APPLE_BUNDLE_ID=
ENV APPLE_ISSUER_ID=
ENV APPLE_KEY_ID=
ENV APPLE_PRIVATE_KEY=

# ========== Google IAP Configuration ==========
ENV GOOGLE_PACKAGE_NAME=
ENV GOOGLE_SERVICE_ACCOUNT_KEY=

EXPOSE 8081
CMD ["/app/grapery-vippay"]

# ========================================
# Default target is server
# ========================================
FROM server
