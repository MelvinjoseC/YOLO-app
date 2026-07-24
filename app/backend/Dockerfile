# ==========================================
# Build Stage
# ==========================================
FROM golang:1.20-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Leverage Docker cache for dependencies
COPY go.mod ./
RUN go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /yolo-api .

# ==========================================
# Run Stage
# ==========================================
FROM alpine:3.18.2

# Setup secure environment
RUN apk add --no-cache ca-certificatestzdata \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /yolo-api /app/yolo-api

# Use non-root user for container security hardening
USER appuser

EXPOSE 8080

ENV PORT=8080

ENTRYPOINT ["/app/yolo-api"]
