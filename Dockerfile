# Указание версии Go и базового образа как аргументов
ARG GO_VERSION
ARG BASE_IMAGE

# Этап 1: Сборка
FROM golang:${GO_VERSION} AS builder

WORKDIR /app

# Установка зависимостей для CGO (необходим gcc для SQLite)
RUN apt-get update && apt-get install -y gcc libsqlite3-dev

# Копирование файлов модулей и загрузка зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Установка флагов сборки для CGO
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o literary-jo main.go

# Этап 2: Окончательная среда выполнения
FROM ${BASE_IMAGE}
WORKDIR /app

# Установка сертификатов и необходимых библиотек для совместимости
RUN if [ "${BASE_IMAGE}" = "alpine" ]; then \
      apk --no-cache add ca-certificates sqlite-libs; \
    elif [ "${BASE_IMAGE}" = "debian" ] || [ "${BASE_IMAGE}" = "ubuntu" ]; then \
      apt-get update && apt-get install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*; \
    fi

# Копирование скомпилированного бинарного файла и необходимых ресурсов
COPY --from=builder /app/literary-jo .
COPY --from=builder /app/assets ./assets
COPY --from=builder /app/internal/db/forum.db ./internal/db/forum.db

EXPOSE 8080
CMD ["./literary-jo"]
