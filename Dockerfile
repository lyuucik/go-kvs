# 🔧 1. Берем официальный образ Go для сборки (builder)
FROM golang:1.24-alpine AS builder

# 🚨 Важно: alpine легче, чем debian-based образы
# Если нужны C-бинды (например, для SQLite), то лучше golang:1.21 (но он тяжелее)

# 2. Создаем рабочую директорию
WORKDIR /app

# 3. Копируем зависимости отдельно (чтобы кэшировать слои)
COPY ../go.mod ../go.sum ./
RUN go mod download

# 🚨 Если у тебя нет go.sum – удали эту строку, но лучше сделай `go mod tidy`!

# 4. Копируем исходники
COPY . .

# 5. Собираем бинарник (статически линкуем, чтобы работал в scratch)
RUN ls
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" /app/src/cmd/api/main.go

# 🔧 6. Переносим бинарник в чистый образ (scratch или alpine)
FROM alpine:3.18

COPY --from=builder /app/main /app

ENTRYPOINT ["/app"]