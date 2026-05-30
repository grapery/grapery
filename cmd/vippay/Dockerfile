# syntax=docker/dockerfile:1

# ========================================
# Dockerfile for Grapery VIPPay Service
# (VIP Payment Service)
# ========================================

FROM golang:1.25 AS builder
WORKDIR /src

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . ./

# Build the vippay binary
ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.CommitSHA=${COMMIT_SHA} -X main.BuildTime=${BUILD_TIME}" \
    -o /bin/grapery-vippay ./cmd/vippay

# ========================================
# Production Image
# ========================================
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /bin/grapery-vippay /app/grapery-vippay
# Copy config file (optional, service will fall back to env vars if file is missing)
COPY --from=builder /src/cmd/vippay/vippay.json /app/vippay.json

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

# ========== Apple IAP (StoreKit 2 / App Store Server API) ==========
# GitHub Actions Variables: APPLE_BUNDLE_ID, APPLE_ISSUER_ID, APPLE_KEY_ID, APPLE_PRIVATE_KEY
ENV APPLE_BUNDLE_ID=
ENV APPLE_ISSUER_ID=
ENV APPLE_KEY_ID=
ENV APPLE_PRIVATE_KEY=
ENV IAP_ALLOW_STOREKIT_LOCAL=
# Apple Sandbox overrides (optional; default = production values above)
ENV APPLE_SANDBOX_BUNDLE_ID=
ENV APPLE_SANDBOX_ISSUER_ID=
ENV APPLE_SANDBOX_KEY_ID=
ENV APPLE_SANDBOX_PRIVATE_KEY=

# ========== Google IAP Configuration ==========
ENV GOOGLE_PACKAGE_NAME=
ENV GOOGLE_SERVICE_ACCOUNT_KEY=
# Google Sandbox Configuration (optional, falls back to production values)
ENV GOOGLE_SANDBOX_PACKAGE_NAME=
ENV GOOGLE_SANDBOX_SERVICE_ACCOUNT_KEY=

# ========== Apple OAuth Configuration ==========
ENV APPLE_OAUTH_CLIENT_ID=
ENV APPLE_OAUTH_TEAM_ID=
ENV APPLE_OAUTH_KEY_ID=
ENV APPLE_OAUTH_PRIVATE_KEY=

# ========== Google OAuth Configuration ==========
# Note: Code uses GOOGLE_CLIENT_ID (not GOOGLE_OAUTH_CLIENT_ID)
# Set GOOGLE_CLIENT_ID to Web application Client ID (Server Client ID)
ENV GOOGLE_CLIENT_ID=
ENV GOOGLE_OAUTH_CLIENT_SECRET=

# ========== APNs Push Notification Configuration ==========
ENV APNS_BUNDLE_ID=
ENV APNS_TEAM_ID=
ENV APNS_KEY_ID=
ENV APNS_PRIVATE_KEY=
ENV APNS_USE_SANDBOX=false

# ========== FCM Push Notification Configuration ==========
ENV FCM_PROJECT_ID=
ENV FCM_CREDENTIALS_JSON=

# ========== Aliyun SLS Telemetry Configuration ==========
ENV ALIYUN_ACCESS_KEY_ID=
ENV ALIYUN_ACCESS_KEY_SECRET=

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/grapery-vippay", "-version"] || exit 1

CMD ["/app/grapery-vippay"]

