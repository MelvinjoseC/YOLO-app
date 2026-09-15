# ==========================================
# Build Stage
# ==========================================
FROM golang:1.20-alpine AS builder

ARG VERSION=1.0.0
ARG REVISION=unknown

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Leverage Docker cache for dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.Revision=${REVISION}" \
    -o /yolo-api .

# ==========================================
# Run Stage
# ==========================================
FROM alpine:3.18.2

ARG VERSION=1.0.0
ARG REVISION=unknown

LABEL org.opencontainers.image.title="yolo-api" \
      org.opencontainers.image.description="YOLO Cloud-Native REST API Service" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.licenses="Apache-2.0"

# Setup secure environment with required certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /yolo-api /app/yolo-api

# Use non-root user for container security hardening
USER appuser

EXPOSE 8080

ENV PORT=8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/livez || exit 1

ENTRYPOINT ["/app/yolo-api"]
