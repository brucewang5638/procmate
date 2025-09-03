#!/bin/bash

# === 步骤 0: 定义程序源路径 ===
# 使用 ${1:-.} 语法：
# - 如果用户提供了第一个参数 (./install.sh /some/path)，则 PROCMATE_SOURCE_PATH 的值为 /some/path
# - 如果用户未提供参数 (./install.sh)，则 PROCMATE_SOURCE_PATH 的值为 . (当前目录)
PROCMATE_SOURCE_PATH="${1:-.}"
PROCMATE_BINARY_PATH="${PROCMATE_SOURCE_PATH}/procmate"
PROCMATE_CONFIG_PATH="${PROCMATE_SOURCE_PATH}/config.yaml"
PROCMATE_CONFD_PATH="${PROCMATE_SOURCE_PATH}/procmate.d"

# 检查 procmate 文件是否存在于指定路径
if [ ! -f "${PROCMATE_BINARY_PATH}" ]; then
    echo "错误: 在路径 '${PROCMATE_BINARY_PATH}' 下找不到 'procmate' 可执行文件。"
    exit 1
fi

# 检查 config.yaml 文件是否存在于指定路径
if [ ! -f "${PROCMATE_CONFIG_PATH}" ]; then
    echo "错误: 在路径 '${PROCMATE_CONFIG_PATH}' 下找不到 'config.yaml' 配置文件。"
    exit 1
fi

# <-- 检查 procmate.d 目录是否存在 -->
if [ ! -d "${PROCMATE_CONFD_PATH}" ]; then
    echo "错误: 在路径 '${PROCMATE_CONFD_PATH}' 下找不到 'procmate.d' 配置目录。"
    exit 1
fi

# === 步骤 1: 安装 procmate 二进制文件 ===
echo "正在安装 procmate 程序..."
sudo mkdir -p /opt/procmate
# 使用 cp 代替 mv，这在安装脚本中是更常见的做法
sudo cp "${PROCMATE_BINARY_PATH}" /opt/procmate/
sudo chmod 755 /opt/procmate/procmate
sudo ln -sf /opt/procmate/procmate /usr/local/bin/procmate
echo "✅ 程序已安装!"
echo ""

# === 步骤 2: 安装配置文件 ===
sudo mkdir -p /etc/procmate
echo "正在复制配置文件..."
# 使用 cp 代替 mv
sudo cp "${PROCMATE_CONFIG_PATH}" /etc/procmate/
# <-- 新增：递归复制 procmate.d 目录 -->
sudo cp -r "${PROCMATE_CONFD_PATH}" /etc/procmate/
echo "✅ 默认配置文件已创建于 /etc/procmate/"
echo ""

echo "🎉 procmate 安装与配置完成！现在您可以在任何目录下运行 'procmate' 查看帮助。"