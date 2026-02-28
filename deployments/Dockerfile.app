# Dockerfile.app - Multi-stage build for the Admin PWA (cmd/app)
#
# Stage 1: Build Wasm frontend (GOOS=js GOARCH=wasm)
# Stage 2: Build native server binary
# Stage 3: Minimal runtime image

# ---- Stage 1: Build Wasm frontend ----
FROM golang:1.26 AS wasm-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -trimpath -o /out/app.wasm ./cmd/app

# ---- Stage 2: Build server binary ----
FROM golang:1.24 AS server-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w \
      -X github.com/flowline-io/flowbot/version.Buildstamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ') \
      -X github.com/flowline-io/flowbot/version.Buildtags=$(git describe --tags --always 2>/dev/null || echo unknown)" \
    -o /out/app-server ./cmd/app

# ---- Stage 3: Runtime ----
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /opt/app

COPY --from=server-builder /out/app-server .
COPY --from=wasm-builder /out/app.wasm ./web/app.wasm

RUN chmod +x app-server

EXPOSE 8090

ENTRYPOINT ["/opt/app/app-server"]
