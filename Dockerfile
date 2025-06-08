FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/bot


FROM alpine:latest

# Установка зависимостей: curl, jq, и kubectl
RUN apk --no-cache add curl jq
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
    rm kubectl

# Установка Yandex Cloud CLI
RUN curl -sSL https://storage.yandexcloud.net/yandexcloud-yc/install.sh | \
    sh -s -- -i /usr/local/yandex-cloud -n

ENV PATH="/root/.yandex-cloud/bin:${PATH}"

WORKDIR /app

COPY --from=builder /app/main .

# Копируем ключ сервисного аккаунта и скрипт запуска
COPY key.json .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

# Запускаем скрипт
CMD ["./entrypoint.sh"]

