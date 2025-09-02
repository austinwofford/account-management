# syntax=docker/dockerfile:1.7

############################
# Build (static binary)
############################
FROM golang:1.25.0-alpine AS build
WORKDIR /src

# Certs for HTTPS egress during build
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
    -o /out/account-management ./cmd/account-management

############################
# Run (distroless, non-root)
############################
FROM gcr.io/distroless/static:nonroot
# CA bundle for outbound TLS
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# App binary
COPY --from=build /out/account-management /account-management

USER nonroot
EXPOSE 8080
ENTRYPOINT ["/account-management"]
