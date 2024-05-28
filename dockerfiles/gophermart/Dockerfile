FROM golang:1.21.3-alpine AS builder

# Создаем и переходим в директорию приложения.
WORKDIR /app

# Копируем go.mod и go.sum для загрузки зависимостей.
COPY go.mod go.sum ./

# Загружаем все зависимости. Зависимости будут кэшированы, если файлы go.mod и go.sum не были изменены.
RUN go mod download

# Копируем исходный код из текущего каталога в рабочий каталог внутри контейнера.
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o gophermart ./cmd/gophermart/main.go

FROM alpine:3.19

# перемещаем исполняемый и другие файлы в нужную директорию
WORKDIR /app/

COPY --from=builder --chown=app:app app .

CMD ["./gophermart"]