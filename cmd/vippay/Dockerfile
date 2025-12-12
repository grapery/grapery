# syntax=docker/dockerfile:1

# ========================================
# Dockerfile for Grapery VIPPay Service
# (VIP Payment Service)
# ========================================

FROM golang:1.24 AS builder
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

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/grapery-vippay", "-version"] || exit 1

CMD ["/app/grapery-vippay"]

