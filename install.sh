#!/bin/bash

# === 步骤 0: 定义程序源路径 ===
# 使用 ${1:-.} 语法：
# - 如果用户提供了第一个参数 (./install.sh /some/path)，则 PROCMATE_SOURCE_PATH 的值为 /some/path
# - 如果用户未提供参数 (./install.sh)，则 PROCMATE_SOURCE_PATH 的值为 . (当前目录)
PROCMATE_SOURCE_PATH="${1:-.}"
PROCMATE_BINARY_PATH="${PROCMATE_SOURCE_PATH}/procmate"
PROCMATE_CONFIG_PATH="${PROCMATE_SOURCE_PATH}/config.yaml"

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

# === 步骤 : 安装 procmate 二进制文件 ===
echo "正在安装 procmate 程序..."
sudo mkdir -p /opt/procmate
# 使用变量来移动指定路径下的文件
sudo mv "${PROCMATE_BINARY_PATH}" /opt/procmate/
sudo chmod 755 /opt/procmate/procmate
sudo ln -sf /opt/procmate/procmate /usr/local/bin/procmate
echo "✅ 程序已安装!"
echo ""

# === 步骤 : 安装配置文件 ===
sudo mkdir -p /etc/procmate
echo "正在移动配置文件..."
sudo mv "${CONFIG_SOURCE_PATH}" /etc/procmate/
echo "✅ 默认配置文件已创建于 /etc/procmate/config.yaml"
echo ""

echo "🎉 procmate 安装与配置完成！现在您可以在任何目录下运行 'procmate' 查看帮助。"