#!/bin/bash
set -e

# 获取当前脚本的绝对目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 引用通用变量及脚本
source "$SCRIPT_DIR/services.sh"
source "$SCRIPT_DIR/../entity-utils.sh"

PIDFILE="/run/hk-watch-services.pid"

# 读取想要停止的服务名
service_name_input="$1"

# ✅ 所有合法服务名
all_service_names=("${services[@]}")

# ✅ 校验参数
if [ -z "$service_name_input" ] || { [ "$service_name_input" != "all" ] && [[ ! " ${all_service_names[*]} " =~ " ${service_name_input} " ]]; }; then
  echo "❌ 无效服务名：${service_name_input:-<未指定>}"
  echo "✅ 用法: ./stop-service-all.sh [服务名 | all]"
  echo "📋 可用服务名："
  for name in "${all_service_names[@]}"; do
    echo "  - $name"
  done
  echo "  - all   # 停止所有服务"
  exit 1
fi

# ✅ 如是 all，则优先终止 watch 脚本（防止它重启组件）
if [ "$service_name_input" == "all" ]; then
  echo "🛑 尝试停止正在运行的 watch-services.sh 守护脚本..."
  if [ -f "$PIDFILE" ]; then
    PID=$(cat "$PIDFILE")
    kill "$PID" && echo "✅ 守护进程 watch-services.sh 已终止"
    rm -f "$PIDFILE"
  else
    echo "⚠️ 守护进程 PID 文件不存在，可能未启动"
  fi
fi

# ✅ 构造 grep 用的正则表达式  例如：smzjg-order|smzjg-report|smzjg-assets ...
if [ "$service_name_input" == "all" ]; then
  pattern=$(IFS='|'; echo "${all_service_names[*]}")
else
  pattern="$service_name_input"
fi

#  将 - 替换为 . 以适配 Java 启动参数（如 -Dname=xxx） 例如：smzjg.order|smzjg.report|smzjg.assets ...
pattern=$(echo "$pattern" | sed 's/-/./g')  

# 3️⃣ 构造匹配 Java 启动进程的 grep 正则表达式 例如：java ... -Dname=smzjg.order ...
stop_entities_by_pattern "java .*($pattern)" 30
