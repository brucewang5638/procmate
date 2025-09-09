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
PROCMATE_SOURCE_CONFD="${PROCMATE_SOURCE_PATH}/conf.d"

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

# === 步骤 3: 安装二进制文件 ===
echo "🚀 正在安装 procmate 程序..."
sudo mkdir -p "${PROCMATE_INSTALL_DIR}"
sudo cp "${PROCMATE_BINARY_PATH}" "${PROCMATE_INSTALL_DIR}/"
sudo chmod 755 "${PROCMATE_INSTALL_DIR}/procmate"
sudo ln -sf "${PROCMATE_INSTALL_DIR}/procmate" "${PROCMATE_BIN_LINK}"
echo "✅ 程序已安装!"
echo ""

# === 步骤 4: 安装配置文件 ===
echo "📦 正在复制配置文件..."
sudo mkdir -p "${PROCMATE_ETC_DIR}"

# --- 智能处理主配置文件 config.yaml ---
TARGET_CONFIG_FILE="${PROCMATE_ETC_DIR}/config.yaml"
if [ -f "${TARGET_CONFIG_FILE}" ] && [ "$FORCE_MODE" = false ]; then
    echo "⚠️  警告: 主配置文件 '${TARGET_CONFIG_FILE}' 已存在。跳过复制。"
    echo "     请手动处理该文件，或使用 '--force' 标志运行安装脚本以强制覆盖。"
else
    if [ -f "${TARGET_CONFIG_FILE}" ]; then
        echo "  -> --force 模式: 正在覆盖主配置文件 '${TARGET_CONFIG_FILE}'..."
    else
        echo "  -> 正在复制主配置文件..."
    fi
    sudo cp "${PROCMATE_CONFIG_PATH}" "${TARGET_CONFIG_FILE}"
fi

# --- 智能处理 conf.d 目录 ---
# 确保 conf.d 目标目录存在
sudo mkdir -p "${PROCMATE_ETC_DIR}/conf.d"
if [ -d "${PROCMATE_SOURCE_CONFD}" ]; then
    echo "  -> 正在从源路径复制 'conf.d' 子配置文件..."
    for SOURCE_CONF_FILE in "${PROCMATE_SOURCE_CONFD}"/*; do
        # 确保我们只处理文件
        if [ -f "${SOURCE_CONF_FILE}" ]; then
            TARGET_CONF_FILE="${PROCMATE_ETC_DIR}/conf.d/$(basename "${SOURCE_CONF_FILE}")"

            if [ -f "${TARGET_CONF_FILE}" ] && [ "$FORCE_MODE" = false ]; then
                echo "⚠️  警告: 目标文件 '${TARGET_CONF_FILE}' 已存在。跳过复制。"
                echo "     请手动处理该文件，或使用 '--force' 标志运行安装脚本以强制覆盖。"
            else
                if [ -f "${TARGET_CONF_FILE}" ]; then
                    echo "     --force 模式: 正在覆盖 '${TARGET_CONF_FILE}'..."
                else
                    echo "     正在复制 '${SOURCE_CONF_FILE}'..."
                fi
                sudo cp "${SOURCE_CONF_FILE}" "${TARGET_CONF_FILE}"
            fi
        fi
    done
    echo "✅ 'conf.d' 内容处理完毕。"
else
    echo "ℹ️  源路径中未找到 'conf.d' 目录，跳过子配置复制。"
fi
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