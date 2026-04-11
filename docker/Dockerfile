FROM --platform=$BUILDPLATFORM node:20-alpine AS web-builder

WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS go-builder

WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG GIT_VERSION=unknown

COPY go.mod go.sum ./
RUN go mod download

COPY controller ./controller
COPY core ./core
COPY main.go ./
COPY web ./web
COPY --from=web-builder /src/web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitVersion=${GIT_VERSION}" \
    -o /out/webssh .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /home/webssh webssh

ENV TZ=Asia/Shanghai
WORKDIR /home/webssh

COPY --from=go-builder /out/webssh /usr/local/bin/webssh

EXPOSE 5032
USER webssh

ENTRYPOINT ["webssh"]
