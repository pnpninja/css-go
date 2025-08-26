# syntax=docker/dockerfile:1
FROM golang:1.23-alpine
WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .
RUN go build -o challenge .

ENTRYPOINT ["/app/challenge"]
