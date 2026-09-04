# Linux 构建配置，可通过命令行覆盖，例如：make linux GOARCH=arm64
APP_NAME ?= suse-oaa-backend
CMD_PATH ?= ./cmd
BUILD_DIR ?= bin
BINARY ?= $(BUILD_DIR)/$(APP_NAME)
FINAL_BINARY ?= $(BUILD_DIR)/OAAbeta
GO ?= go
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0
LDFLAGS ?= -s -w

.PHONY: all linux build clean test vet run help

all: linux ## 编译 Linux 可执行文件（默认目标）

linux: ## 编译 Linux 可执行文件，默认目标架构为 amd64
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD_PATH)
	@if [ "$(BINARY)" != "$(FINAL_BINARY)" ]; then mv -f "$(BINARY)" "$(FINAL_BINARY)"; fi
	@echo "已生成: $(FINAL_BINARY)"

build: linux ## linux 的别名

clean: ## 删除构建产物
	@rm -rf $(BUILD_DIR)

test: ## 运行全部测试
	$(GO) test ./...

vet: ## 执行 go vet
	$(GO) vet ./...

run: ## 使用本地配置启动服务
	$(GO) run $(CMD_PATH)

help: ## 显示可用命令
	@awk 'BEGIN {FS = ":.*##"; printf "用法: make <目标> [变量=值]\n\n目标:\n"} /^[a-zA-Z_%-]+:.*##/ { printf "  %-10s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
