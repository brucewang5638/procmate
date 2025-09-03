#!/bin/bash

# === 步骤 0: 定义程序源路径 ===
# 使用 ${1:-.} 语法：
# - 如果用户提供了第一个参数 (./install.sh /some/path)，则 PROCMATE_SOURCE_PATH 的值为 /some/path
# - 如果用户未提供参数 (./install.sh)，则 PROCMATE_SOURCE_PATH 的值为 . (当前目录)
PROCMATE_SOURCE_PATH="${1:-.}"
PROCMATE_BINARY_PATH="${PROCMATE_SOURCE_PATH}/procmate"
PROCMATE_CONFIG_PATH="${PROCMATE_SOURCE_PATH}/config.yaml"
PROCMATE_SERVICE_PATH="${PROCMATE_SOURCE_PATH}/release-scripts/procmate.service" # <-- 新增

# === 统一定义安装目标路径 ===
PROCMATE_INSTALL_DIR="/opt/procmate"
PROCMATE_BIN_LINK="/usr/local/bin/procmate"
PROCMATE_ETC_DIR="/etc/procmate"
PROCMATE_SERVICE_TARGET="/etc/systemd/system/procmate.service"

# === 步骤 0: 文件检查 ===
if [ ! -f "${PROCMATE_BINARY_PATH}" ]; then
    echo "错误: 在路径 '${PROCMATE_BINARY_PATH}' 下找不到 'procmate' 可执行文件。"
    exit 1
fi

if [ ! -f "${PROCMATE_CONFIG_PATH}" ]; then
    echo "错误: 在路径 '${PROCMATE_CONFIG_PATH}' 下找不到 'config.yaml' 配置文件。"
    exit 1
fi

if [ ! -f "${PROCMATE_SERVICE_PATH}" ]; then
    echo "错误: 在路径 '${PROCMATE_SERVICE_PATH}' 下找不到 'procmate.service' 服务文件。"
    exit 1
fi

# === 步骤 1: 安装二进制 ===
echo "正在安装 procmate 程序..."
sudo mkdir -p "${PROCMATE_INSTALL_DIR}"
sudo cp "${PROCMATE_BINARY_PATH}" "${PROCMATE_INSTALL_DIR}/"
sudo chmod 755 "${PROCMATE_INSTALL_DIR}/procmate"
sudo ln -sf "${PROCMATE_INSTALL_DIR}/procmate" "${PROCMATE_BIN_LINK}"
echo "✅ 程序已安装!"
echo ""

# === 步骤 2: 安装配置文件 ===
echo "正在复制配置文件..."
sudo mkdir -p "${PROCMATE_ETC_DIR}"
sudo cp "${PROCMATE_CONFIG_PATH}" "${PROCMATE_ETC_DIR}/"
echo "✅ 默认主配置文件已创建于 ${PROCMATE_ETC_DIR}/"
# 确保 procmate.d 目录存在
if [ ! -d "${PROCMATE_ETC_DIR}/procmate.d" ]; then
    sudo mkdir -p "${PROCMATE_ETC_DIR}/procmate.d"
    echo "✅ 默认子配置文件目录已创建于 ${PROCMATE_ETC_DIR}/procmate.d"
else
    echo "ℹ️ 已存在 ${PROCMATE_ETC_DIR}/procmate.d，跳过创建。"
fi
echo ""

# === 步骤 3: 安装 systemd 服务 ===
echo "正在设置 systemd 服务..."
sudo cp "${PROCMATE_SERVICE_PATH}" "${PROCMATE_SERVICE_TARGET}"
sudo systemctl daemon-reload
sudo systemctl enable procmate
echo "✅ procmate 服务已启用，将在下次启动时自动运行。"
echo ""
echo "您现在可以手动启动服务: sudo systemctl start procmate"
echo "或查看服务状态: sudo systemctl status procmate"
echo ""

echo "🎉 procmate 安装与配置完成！"