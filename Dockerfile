FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o file-service cmd/file-service/main.go

FROM alpine:latest

COPY --from=builder /app/file-service /app/file-service

COPY migrations /app/migrations

WORKDIR /app

CMD ["./file-service"]