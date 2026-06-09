#!/bin/bash

# 任何命令失败则立即退出
set -e

# === 步骤 1: 解析参数 ===

PURGE_MODE=false

for arg in "$@"; do
  case "$arg" in
    -p|--purge)
      PURGE_MODE=true
      ;;
    *)
      echo "❌ 未知参数: $arg"
      echo "用法: ./uninstall.sh [--purge]"
      echo ""
      echo "  --purge, -p    同时删除 /etc/procmate 配置目录"
      exit 1
      ;;
  esac
done

# === 步骤 2: 定义路径 ===

PROCMATE_INSTALL_DIR="/opt/procmate"
PROCMATE_BIN_LINK="/usr/local/bin/procmate"
PROCMATE_ETC_DIR="/etc/procmate"
PROCMATE_SERVICE_TARGET="/etc/systemd/system/procmate.service"

echo "🧹 开始卸载 procmate..."
echo ""

# === 步骤 3: 停止并禁用 systemd 服务 ===

echo "🛑 正在停止 procmate 服务..."

if systemctl list-unit-files | grep -q "^procmate.service"; then
    if systemctl is-active --quiet procmate; then
        sudo systemctl stop procmate
        echo "✅ procmate 服务已停止。"
    else
        echo "ℹ️  procmate 服务当前未运行。"
    fi

    if systemctl is-enabled --quiet procmate 2>/dev/null; then
        sudo systemctl disable procmate
        echo "✅ procmate 服务已禁用开机自启。"
    else
        echo "ℹ️  procmate 服务未设置开机自启。"
    fi
else
    echo "ℹ️  未发现 procmate systemd 服务。"
fi

echo ""

# === 步骤 4: 删除 systemd 服务文件 ===

echo "🛠️  正在删除 systemd 服务文件..."

if [ -f "${PROCMATE_SERVICE_TARGET}" ]; then
    sudo rm -f "${PROCMATE_SERVICE_TARGET}"
    sudo systemctl daemon-reload
    sudo systemctl reset-failed procmate 2>/dev/null || true
    echo "✅ systemd 服务文件已删除。"
else
    echo "ℹ️  未找到服务文件 '${PROCMATE_SERVICE_TARGET}'，跳过。"
fi

echo ""

# === 步骤 5: 删除命令软链接 ===

echo "🔗 正在删除命令软链接..."

if [ -L "${PROCMATE_BIN_LINK}" ] || [ -f "${PROCMATE_BIN_LINK}" ]; then
    sudo rm -f "${PROCMATE_BIN_LINK}"
    echo "✅ 已删除 '${PROCMATE_BIN_LINK}'。"
else
    echo "ℹ️  未找到 '${PROCMATE_BIN_LINK}'，跳过。"
fi

echo ""

# === 步骤 6: 删除安装目录 ===

echo "📦 正在删除程序安装目录..."

if [ -d "${PROCMATE_INSTALL_DIR}" ]; then
    sudo rm -rf "${PROCMATE_INSTALL_DIR}"
    echo "✅ 已删除 '${PROCMATE_INSTALL_DIR}'。"
else
    echo "ℹ️  未找到 '${PROCMATE_INSTALL_DIR}'，跳过。"
fi

echo ""

# === 步骤 7: 可选删除配置目录 ===

if [ "$PURGE_MODE" = true ]; then
    echo "⚠️  检测到 '--purge' 参数，将删除配置目录..."

    if [ -d "${PROCMATE_ETC_DIR}" ]; then
        sudo rm -rf "${PROCMATE_ETC_DIR}"
        echo "✅ 已删除配置目录 '${PROCMATE_ETC_DIR}'。"
    else
        echo "ℹ️  未找到配置目录 '${PROCMATE_ETC_DIR}'，跳过。"
    fi
else
    echo "ℹ️  默认保留配置目录 '${PROCMATE_ETC_DIR}'。"
    echo "   如需删除配置，请执行: ./uninstall.sh --purge"
fi

echo ""
echo "🎉 procmate 卸载完成！"
