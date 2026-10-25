#!/usr/bin/env bash
# ==============================================================================
# SUSE OAA 后端中台 - 实体机自动部署脚本 (deploy.sh)
# 用途: 自动从 GitHub Release 下载指定或最新版本二进制制品并安全重启服务
# ==============================================================================

set -euo pipefail

REPO="suse-edu-cn/SUSE-OAA-BACKEND"
APP_NAME="OAAbeta"
if [ -z "${INSTALL_DIR:-}" ]; then
  if [ -d "${HOME}/OAA" ]; then
    INSTALL_DIR="${HOME}/OAA"
  else
    INSTALL_DIR="$(pwd)/bin"
  fi
fi
SERVICE_NAME="${SERVICE_NAME:-suse-oaa}"
TMP_DIR=$(mktemp -d)

trap 'rm -rf "${TMP_DIR}"' EXIT

# 1. 确定目标版本与制品来源
TAG="${1:-}"
LOCAL_PACKAGE="${2:-}"

if [ -z "${TAG}" ]; then
  echo "==> 未指定版本号，正在查询最新 Release..."
  TAG=$(curl -s --connect-timeout 10 "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "${TAG}" ]; then
    echo "❌ 获取最新 Release 标签失败，请检查网络或传入具体版本号 (例如: ./deploy.sh v2.0.1)"
    exit 1
  fi
fi

TAR_NAME="suse-oaa-backend-${TAG}-linux-amd64.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${TAR_NAME}"

echo "==> 准备部署版本: ${TAG}"
echo "==> 目标安装目录: ${INSTALL_DIR}"

# 2. 获取制品包 (多镜像智能加速下载)
mkdir -p "${INSTALL_DIR}"
echo "==> 准备获取制品包..."

download_package() {
  local target_file="${TMP_DIR}/${TAR_NAME}"

  # 1. 若本地已传入制品文件且存在，直接使用
  if [ -n "${LOCAL_PACKAGE:-}" ] && [ -f "${LOCAL_PACKAGE}" ]; then
    echo "==> 检测到本地已存在制品包: ${LOCAL_PACKAGE}，直接使用本地制品..."
    cp -f "${LOCAL_PACKAGE}" "${target_file}"
    return 0
  fi

  # 2. 依次尝试下载：优先官方源（通常十几秒），若网络波动则降级至国内加速镜像
  echo "==> 准备下载制品包..."
  MIRRORS=(
    "${DOWNLOAD_URL}"
    "https://ghfast.top/${DOWNLOAD_URL}"
    "https://gh-proxy.com/${DOWNLOAD_URL}"
  )

  for mirror in "${MIRRORS[@]}"; do
    echo "==> 尝试下载源: ${mirror}"
    # 放宽连接超时至 20s（适应跨国 DNS 与 302 S3 双重 TLS 握手前摇），单源总时限 90s
    if curl --connect-timeout 20 --max-time 90 -fL -o "${target_file}" "${mirror}"; then
      echo "✅ 下载成功！"
      return 0
    fi
    echo "⚠️ 当前源失败，尝试下一个..."
  done

  echo "❌ 所有下载渠道均已尝试，下载制品包失败！"
  return 1
}

download_package

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
TMUX_SESSION="OAA"

if command -v tmux >/dev/null 2>&1 && tmux has-session -t "${TMUX_SESSION}" 2>/dev/null; then
  echo "==> 检测到 tmux 会话 [${TMUX_SESSION}]，发送 Ctrl+C 触发优雅停机..."
  tmux send-keys -t "${TMUX_SESSION}" C-c
  sleep 2
  echo "==> 在 tmux 会话 [${TMUX_SESSION}] 中启动新版 ${APP_NAME}..."
  tmux send-keys -t "${TMUX_SESSION}" "cd ${INSTALL_DIR} && ./${APP_NAME}" Enter
  echo "✅ 已在 tmux [${TMUX_SESSION}] 窗口中成功重启最新版本！"
elif command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
  echo "==> 通过 systemctl 重启 ${SERVICE_NAME} 服务..."
  sudo systemctl restart "${SERVICE_NAME}"
  echo "✅ ${SERVICE_NAME} 服务已平滑重启！"
else
  echo "ℹ️ 未检测到运行中的 tmux 会话 [${TMUX_SESSION}] 或 systemd 服务 [${SERVICE_NAME}]。"
  echo "   已更新二进制产物至: ${INSTALL_DIR}/${APP_NAME}"
  echo "   若需手动在 tmux 启动，可执行: tmux attach -t ${TMUX_SESSION}"
fi

echo "=============================================================================="
echo "🎉 部署完成！当前运行版本: ${TAG}"
echo "=============================================================================="
