.PHONY: web build release clean

# 构建前端并同步到嵌入目录
web:
	cd web && pnpm install --no-frozen-lockfile && pnpm build
	rm -rf ui/dist
	cp -r web/dist ui/dist

# 本机构建（含前端）
build: web
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/token-proxy .

# 交叉编译发布产物（Linux amd64/arm64 + macOS arm64），并打成 tar.gz
release: web
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/token-proxy-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/token-proxy-linux-arm64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/token-proxy-darwin-arm64 .
	cd dist && tar czf token-proxy-linux-amd64.tar.gz token-proxy-linux-amd64 && tar czf token-proxy-linux-arm64.tar.gz token-proxy-linux-arm64 && rm -f token-proxy-linux-amd64 token-proxy-linux-arm64 token-proxy-darwin-arm64
	@echo "===== 发布产物 =====" && ls -lh dist

clean:
	rm -rf bin dist web/dist ui/dist/*
	touch ui/dist/index.html