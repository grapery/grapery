# syntax=docker/dockerfile:1

# ========================================
# Dockerfile for Grapery Server Service
# ========================================

FROM golang:1.24 AS builder
WORKDIR /src

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . ./

# Build the server binary
ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.CommitSHA=${COMMIT_SHA} -X main.BuildTime=${BUILD_TIME}" \
    -o /bin/grapery-server ./cmd/server

# ========================================
# Production Image
# ========================================
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /bin/grapery-server /app/grapery-server

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

# ========== Aliyun SLS Telemetry Configuration ==========
ENV ALIYUN_ACCESS_KEY_ID=
ENV ALIYUN_ACCESS_KEY_SECRET=

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/grapery-server", "-version"] || exit 1

CMD ["/app/grapery-server"]

