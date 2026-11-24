# syntax=docker/dockerfile:1

# ---------- 1. 构建前端 ----------
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json ./
RUN npm install --registry=https://registry.npmmirror.com
COPY web/ ./
RUN npm run build

# ---------- 2. 构建后端（嵌入前端） ----------
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./ui/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/token-proxy .

# ---------- 3. 运行镜像 ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /opt/token-proxy
COPY --from=build /out/token-proxy /opt/token-proxy/token-proxy
ENV TOKEN_PROXY_LISTEN=:3000 \
    TOKEN_PROXY_DATA_DIR=/data
VOLUME /data
EXPOSE 3000
ENTRYPOINT ["/opt/token-proxy/token-proxy"]