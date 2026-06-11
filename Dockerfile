FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/server ./cmd/server

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/simulator ./cmd/data_generator

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

COPY --from=builder /app/server /app/server
COPY --from=builder /app/simulator /app/simulator
COPY --from=builder /app/config.yaml /app/config.yaml
COPY --from=builder /app/web /app/web
COPY --from=builder /app/sql /app/sql

EXPOSE 8080 6060 9090

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/api/statistics || exit 1

ENTRYPOINT ["/app/server"]
CMD ["-config", "/app/config.yaml", "-port", "8080"]
