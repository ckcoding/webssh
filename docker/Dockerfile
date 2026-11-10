# syntax=docker/dockerfile:1
#
# Build:  docker build -f docker/Dockerfile -t webssh:local .
# Multi-arch: docker buildx build --platform linux/amd64,linux/arm64 -f docker/Dockerfile -t webssh:local .

# ---- 1. Frontend -------------------------------------------------------------
FROM --platform=$BUILDPLATFORM node:20-alpine AS web-builder

ARG NPM_REGISTRY=https://registry.npmjs.org/
WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm config set registry "$NPM_REGISTRY" \
    && npm ci --no-audit --no-fund

COPY web/ ./
RUN npm run build

# ---- 2. Backend (frontend is embedded via go:embed) -------------------------
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS go-builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG GIT_VERSION=unknown
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org

ENV GOPROXY=$GOPROXY \
    GOSUMDB=$GOSUMDB \
    GOTOOLCHAIN=local \
    CGO_ENABLED=0

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY controller ./controller
COPY core ./core
COPY main.go ./
COPY --from=web-builder /src/web/dist ./web/dist

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitVersion=${GIT_VERSION}" \
    -o /out/webssh .

# ---- 3. Runtime --------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 webssh \
    && adduser -S -D -u 10001 -G webssh -h /home/webssh webssh \
    && mkdir -p /home/webssh/.config/webssh \
    && chown -R webssh:webssh /home/webssh

COPY --from=go-builder /out/webssh /usr/local/bin/webssh

ENV TZ=Asia/Shanghai \
    port=5032 \
    savePass=true

WORKDIR /home/webssh
USER webssh
EXPOSE 5032
VOLUME ["/home/webssh/.config/webssh"]

# Follows the runtime port / basePath env vars; /api/status needs no login.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- "http://127.0.0.1:${port:-5032}${basePath:-${BASE_PATH:-}}/api/status" >/dev/null || exit 1

ENTRYPOINT ["webssh"]
