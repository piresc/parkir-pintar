# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build args injected at build time
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Build all service binaries
RUN for svc in gateway reservation search billing payment presence notification; do \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -ldflags="-s -w \
          -X main.Version=${VERSION} \
          -X main.GitCommit=${GIT_COMMIT} \
          -X main.BuildTime=${BUILD_TIME}" \
        -o /build/${svc} ./cmd/${svc}; \
    done

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy all binaries and migrations from builder
COPY --from=builder /build/gateway /build/reservation /build/search /build/billing /build/payment /build/presence /build/notification ./
COPY --from=builder /build/db/migrations ./db/migrations
COPY --from=builder /build/docs/swagger.yaml ./docs/swagger.yaml
COPY --from=builder /build/docs/swagger-ui ./docs/swagger-ui

# Switch to non-root user
USER appuser

EXPOSE 8080
