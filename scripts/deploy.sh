#!/usr/bin/env bash
# ==============================================================================
# SUSE OAA 后端中台 - 实体机自动部署脚本 (deploy.sh)
# 用途: 自动从 GitHub Release 下载指定或最新版本二进制制品并安全重启服务
# ==============================================================================

set -euo pipefail

REPO="suse-edu-cn/SUSE-OAA-BACKEND"
APP_NAME="OAAbeta"
INSTALL_DIR="${INSTALL_DIR:-$(pwd)/bin}"
SERVICE_NAME="${SERVICE_NAME:-suse-oaa}"
TMP_DIR=$(mktemp -d)

trap 'rm -rf "${TMP_DIR}"' EXIT

# 1. 确定目标版本
TAG="${1:-}"
if [ -z "${TAG}" ]; then
  echo "==> 未指定版本号，正在查询最新 Release..."
  TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "${TAG}" ]; then
    echo "❌ 获取最新 Release 标签失败，请检查网络或传入具体版本号 (例如: ./deploy.sh v2.0.1)"
    exit 1
  fi
fi

TAR_NAME="suse-oaa-backend-${TAG}-linux-amd64.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${TAR_NAME}"

echo "==> 准备部署版本: ${TAG}"
echo "==> 目标安装目录: ${INSTALL_DIR}"
echo "==> 下载地址: ${DOWNLOAD_URL}"

# 2. 下载制品包
mkdir -p "${INSTALL_DIR}"
echo "==> 开始下载制品包..."
curl -fL -o "${TMP_DIR}/${TAR_NAME}" "${DOWNLOAD_URL}"

# 3. 解压并校验
echo "==> 正在解压制品..."
tar -zxvf "${TMP_DIR}/${TAR_NAME}" -C "${TMP_DIR}"

if [ ! -f "${TMP_DIR}/${APP_NAME}" ]; then
  echo "❌ 制品中未找到可执行文件 ${APP_NAME}！"
  exit 1
fi

# 4. 备份当前旧版本（如有）
if [ -f "${INSTALL_DIR}/${APP_NAME}" ]; then
  echo "==> 备份现有可执行文件为 ${APP_NAME}.bak..."
  cp -f "${INSTALL_DIR}/${APP_NAME}" "${INSTALL_DIR}/${APP_NAME}.bak"
fi

# 5. 替换新版本二进制并赋予执行权限
echo "==> 覆盖安装新版本二进制..."
mv -f "${TMP_DIR}/${APP_NAME}" "${INSTALL_DIR}/${APP_NAME}"
chmod +x "${INSTALL_DIR}/${APP_NAME}"

# 6. 重启服务
echo "==> 正在触发服务重启..."
if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
  echo "==> 通过 systemctl 重启 ${SERVICE_NAME} 服务..."
  sudo systemctl restart "${SERVICE_NAME}"
  echo "✅ ${SERVICE_NAME} 服务已平滑重启！"
else
  echo "ℹ️ 未检测到运行中的 systemd 服务 [${SERVICE_NAME}]。"
  echo "   已更新二进制产物至: ${INSTALL_DIR}/${APP_NAME}"
  echo "   请按当前主机的守护进程方式（如 nohup / supervisor / systemd）自行重启或加载。"
fi

echo "=============================================================================="
echo "🎉 部署完成！当前运行版本: ${TAG}"
echo "=============================================================================="
