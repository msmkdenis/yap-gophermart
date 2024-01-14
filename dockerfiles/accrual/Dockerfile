FROM golang:1.21.3-alpine3.17

WORKDIR /app

COPY ../.. /app

RUN go mod download

ENTRYPOINT go run cmd/mockaccrual/main.go

# command to be used to execute when the image is used to start a container
CMD [ "/mockaccrual" ]