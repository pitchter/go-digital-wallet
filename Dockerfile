FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /wallet-service ./cmd/api

FROM alpine:3.20

RUN adduser -D -g '' appuser

WORKDIR /app
COPY --from=builder /wallet-service /usr/local/bin/wallet-service

USER appuser
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/wallet-service"]
