FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN apt install protobuf-compiler
RUN protoc --go_out=. --go-grpc_out=. service.proto
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -ldflags="-s -w" -o ./dist/core-regulus

# Финальный образ
FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/app .

EXPOSE 5000
CMD ["sh", "-c", "source .env && ./app"]