FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN apk add --no-cache \
  git \
  curl \
  protobuf \
  build-base

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN protoc --go_out=. --go-grpc_out=. service.proto
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -ldflags="-s -w" -o ./dist/core-regulus

ENV PATH="/go/bin:${PATH}"

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/app .

EXPOSE 5000
CMD ["sh", "-c", "source .env && ./app"]