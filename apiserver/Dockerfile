# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOOS=linux
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o /out/apiserver ./cmd/apiserver

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
COPY --from=build /out/apiserver /usr/local/bin/apiserver
USER app
EXPOSE 8090
ENTRYPOINT ["apiserver"]
