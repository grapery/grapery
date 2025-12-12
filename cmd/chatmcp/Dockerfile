# syntax=docker/dockerfile:1

# ========================================
# Dockerfile for Grapery ChatMCP Service
# (Agent Chat Service)
# ========================================

FROM golang:1.24 AS builder
WORKDIR /src

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . ./

# Build the chatmcp binary
ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.CommitSHA=${COMMIT_SHA} -X main.BuildTime=${BUILD_TIME}" \
    -o /bin/grapery-chatmcp ./cmd/chatmcp

# ========================================
# Production Image
# ========================================
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /bin/grapery-chatmcp /app/grapery-chatmcp

# ========== Server Configuration ==========
ENV GRAPERY_ENV=production
ENV GRAPERY_HTTP_PORT=8082
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

# ========== AI Configuration (Gemini required for chatmcp) ==========
ENV GEMINI_API_KEY=
ENV GEMINI_BASE_URL=

# ========== JWT Configuration ==========
ENV JWT_SECRET=
ENV JWT_EXPIRY_HOURS=24

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/grapery-chatmcp", "-version"] || exit 1

CMD ["/app/grapery-chatmcp"]

