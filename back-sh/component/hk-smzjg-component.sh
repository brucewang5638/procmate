#!/bin/bash

# --- 配置 ---
# 定义脚本所在的根目录，这样无论在哪里执行命令，都能找到依赖的脚本
SCRIPT_BASE_PATH="/opt/hk/smzjg/script/component"
LOG_BASE_PATH="/var/log/hk/smzjg/component"
COMPONENT_PREFIX="hk-smzjg-components" # 所有 systemd 服务都有这个前缀

# --- 加载配置 ---
# source 命令会执行一个脚本，并将其中的变量导入到当前脚本中
source "${SCRIPT_BASE_PATH}/components.sh"

# --- 参数解析 ---
# $1 是第一个参数 (行为, e.g., stop)
# $2 是第二个参数 (目标, e.g., smzjg-framework)
ACTION=$1
TARGET=$2

# 检查是否提供了行为参数
if [ -z "$ACTION" ]; then
    echo "❌ 错误: 请提供一个行为参数。服务会自动守护所以不提供start"
    echo "📌 用法: hk-smzjg-component {status|logs|stop} [component_name]"
    exit 1
fi

# 如果 ACTION 在上述列表中，则强制检查 TARGET 是否存在
#定义哪些 ACTION 必须需要一个 TARGET
ACTIONS_REQUIRING_TARGET=("stop" "log" "logs")

if [[ " ${ACTIONS_REQUIRING_TARGET[*]} " =~ " ${ACTION} " ]]; then
    if [ -z "$TARGET" ]; then
        echo "❌ 错误: 操作 '${ACTION}' 必须提供一个目标名称。"
        echo "📌 用法: hk-smzjg-component ${ACTION} [component_name]"
        echo ""
        echo "✅ 可用的目标名称必须是以下之一:"
        for name in "${components[@]}"; do
            echo "  - $name"
        done
        exit 1
    fi
fi


# 如果 TARGET 存在，则校验其是否合法
if [ -n "$TARGET" ]; then
    if [[ ! " ${components[*]} " =~ " ${TARGET} " ]]; then
        echo "❌ 错误: 无效的目标名称 ' ${TARGET}'"
        echo ""
        echo "✅ 可用的目标名称必须是以下之一:"
        for name in "${components[@]}"; do
            echo "  - $name"
        done
        exit 1
    fi
fi


# --- 命令分发器 ---
# 根据不同的行为参数，调用不同的底层脚本

SYSTEMD_NAME="${COMPONENT_PREFIX}.service"

case "$ACTION" in
  stop)
    echo "正在执行停止操作，目标: ${TARGET}..."
    bash "${SCRIPT_BASE_PATH}/stop-components.sh" "$TARGET"
    ;;

  status)
    echo "正在实时查看 [${TARGET}] 的状态... (按 Ctrl+C 退出)"
    journalctl -u "$SYSTEMD_NAME" -n 50 --no-pager -f
    ;;

  log | logs)
    echo "正在实时查看 [${TARGET}] 的日志... (按 Ctrl+C 退出)"
    TODAY=$(date +%Y-%m-%d)
    LOG_FILE="${LOG_BASE_PATH}/${TARGET}/${TARGET}-${TODAY}.log"
    echo "目标日志文件: ${LOG_FILE}"

    if [ ! -f "$LOG_FILE" ]; then
        echo "错误: 找不到今天的日志文件: ${LOG_FILE}" >&2
        echo "可能原因: 服务今天还没有产生任何日志。" >&2
        exit 1
    fi

    tail -f "$LOG_FILE"
    ;;

  start-force)
    echo "正在执行启动操作，目标: ${TARGET}..."
    bash "${SCRIPT_BASE_PATH}/start-components.sh" "$TARGET"
    ;;

  status-snapshot)
    echo "正在查询状态，目标: ${TARGET}..."
    systemctl status "$SYSTEMD_NAME"
    ;;

  *)
    echo "错误: 不支持的行为 '$ACTION'。"
    echo "用法: hk-smzjg-component {status|logs|stop} [component_name]"
    exit 1
    ;;
esac

# 检查上一个命令的退出码，给出最终反馈
if [ $? -eq 0 ]; then
    echo "✅ 操作 ' ${ACTION} ${TARGET}' 成功完成。"
else
    echo "❌ 操作 ' ${ACTION} ${TARGET}' 执行失败。"
    exit 1
fi