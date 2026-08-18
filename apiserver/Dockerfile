# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
# GIT_SHA / BUILD_TIME：构建期注入版本信息（ARG 需在 FROM 之后声明才对本阶段可见）
ARG GIT_SHA=unknown
ARG BUILD_TIME=unknown
WORKDIR /src
ENV CGO_ENABLED=0 GOOS=linux
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags "-s -w -X db-cockpit/apiserver/internal/version.GitSHA=${GIT_SHA} -X db-cockpit/apiserver/internal/version.BuildTime=${BUILD_TIME}" -o /out/apiserver ./cmd/apiserver

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
COPY --from=build /out/apiserver /usr/local/bin/apiserver
USER app
EXPOSE 8090
ENTRYPOINT ["apiserver"]
