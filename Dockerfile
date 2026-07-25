FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./src/cmd/api/

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /app/server /server
RUN adduser -D -u 1001 -h /server appuser && chown appuser:appuser /server
USER appuser
HEALTHCHECK --interval=10s --timeout=3s --start-period=3s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1
EXPOSE 8080
ENTRYPOINT ["/server"]
