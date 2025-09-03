#!/bin/bash

# 任何命令失败则立即退出，防止不完整的安装
set -e

# === 步骤 1: 解析参数与定义路径 ===

# --- 解析参数 ---
FORCE_MODE=false
SOURCE_PATH_ARG=""

# 遍历所有传入的参数
for arg in "$@"; do
  case "$arg" in
    -f|--force)
      FORCE_MODE=true
      ;;
    *)
      # 将第一个非标志的参数识别为源路径
      if [ -z "$SOURCE_PATH_ARG" ]; then
        SOURCE_PATH_ARG="$arg"
      fi
      ;;
  esac
done

# 如果识别到 --force 标志，则打印提示信息
if [ "$FORCE_MODE" = true ]; then
    echo "ℹ️  检测到 '--force' 标志，将强制覆盖 'conf.d' 中的同名配置文件。"
fi

# --- 定义路径 ---
# 如果用户未提供路径参数，则默认为当前目录 "."
PROCMATE_SOURCE_PATH="${SOURCE_PATH_ARG:-.}"
PROCMATE_BINARY_PATH="${PROCMATE_SOURCE_PATH}/procmate"
PROCMATE_CONFIG_PATH="${PROCMATE_SOURCE_PATH}/config.yaml"
PROCMATE_CONFD_PATH="${PROCMATE_SOURCE_PATH}/procmate.d"

PROCMATE_INSTALL_DIR="/opt/procmate"
PROCMATE_BIN_LINK="/usr/local/bin/procmate"
PROCMATE_ETC_DIR="/etc/procmate"
PROCMATE_SERVICE_TARGET="/etc/systemd/system/procmate.service"

# === 步骤 2: 文件存在性检查 ===
echo "🔎 正在检查所需文件..."
if [ ! -f "${PROCMATE_BINARY_PATH}" ]; then
    echo "❌ 错误: 在路径 '${PROCMATE_BINARY_PATH}' 下找不到 'procmate' 可执行文件。"
    exit 1
fi

if [ ! -f "${PROCMATE_CONFIG_PATH}" ]; then
    echo "❌ 错误: 在路径 '${PROCMATE_CONFIG_PATH}' 下找不到 'config.yaml' 配置文件。"
    exit 1
fi
echo "✅ 文件检查通过。"
echo ""

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

# === 步骤 5: 安装 systemd 服务 ===
echo "🛠️  正在创建并启用 systemd 服务..."

# 使用 Heredoc 将 service 文件内容直接写入目标路径
sudo tee "${PROCMATE_SERVICE_TARGET}" > /dev/null <<EOF
[Unit]
Description=Procmate Process Manager
After=network.target

[Service]
Type=simple
ExecStart=${PROCMATE_BIN_LINK} watch
Restart=on-failure
RestartSec=5s
User=root
LimitNOFILE=150000

[Install]
WantedBy=multi-user.target
EOF

# 重载并启用服务
sudo systemctl daemon-reload
sudo systemctl enable procmate
echo "✅ procmate 服务已启用，将在下次启动时自动运行。"
echo ""
echo "您现在可以运行: procmate --help 获得帮助!"
echo ""

echo "🎉 procmate 安装与配置完成！"