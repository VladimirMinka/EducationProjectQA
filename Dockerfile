# syntax=docker/dockerfile:1

FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates make protobuf protobuf-dev

ENV PATH="/go/bin:${PATH}"

COPY go.mod go.sum ./
RUN go mod download

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11 \
	&& go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1 \
	&& go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.29.0

COPY . .
RUN make generate \
	&& CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /out/server /app/server
COPY --from=web /web/dist /app/web

ENV WEB_DIR=/app/web

EXPOSE 50051 8080
USER nobody
ENTRYPOINT ["/app/server"]
