#!/bin/bash
set -e

# 获取当前脚本的绝对目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 引用通用变量及脚本
source "$SCRIPT_DIR/components.sh"
source "$SCRIPT_DIR/../entity-utils.sh"

PIDFILE="/run/hk-watch-components.pid"


# ✅ 参数解析
component_name_input="$1"         # 第一个参数：组件名或 all

# ✅ 获取所有合法组件名列表（组件名是大小写敏感的 key）
all_component_names=("${components[@]}")

# ✅ 校验参数合法性
if [ -z "$component_name_input" ] || { [ "$component_name_input" != "all" ] && [[ ! " ${all_component_names[*]} " =~ " ${component_name_input} " ]]; }; then
  echo "❌ 无效组件名：${component_name_input:-<未指定>}"
  echo "✅ 用法: ./stop-component-all.sh [组件名 | all]"
  echo "📋 可用组件名："
  for name in "${all_component_names[@]}"; do
    echo "  - $name"
  done
  echo "  - all   # 停止所有组件"
  exit 1
fi

# ✅ 如是 all，则优先终止 watch 脚本（防止它重启组件）
if [ "$component_name_input" == "all" ]; then
  echo "🛑 尝试停止正在运行的 watch-components.sh 守护脚本..."
  if [ -f "$PIDFILE" ]; then
    PID=$(cat "$PIDFILE")
    kill "$PID" && echo "✅ 守护进程 watch-components.sh 已终止"
    rm -f "$PIDFILE"
  else
    echo "⚠️ 守护进程 PID 文件不存在，可能未启动"
  fi
fi

# ✅ 构造 grep 用正则表达式（匹配组件路径）
if [ "$component_name_input" == "all" ]; then
  pattern=$(IFS='|'; echo "${all_component_names[*]}" | tr '[:upper:]' '[:lower:]')
else
  pattern=$(echo "$component_name_input" | tr '[:upper:]' '[:lower:]')
fi

pattern="$COMPONENT_BASE_PATH/($pattern)"

# ✅ 执行停止操作
echo "🛑 正在停止组件进程匹配：$pattern"
stop_entities_by_pattern "$pattern" "$wait_time"
