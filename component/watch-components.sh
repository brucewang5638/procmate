#!/bin/bash
set -e

# 获取当前脚本的绝对目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 引用通用变量及脚本
source "$SCRIPT_DIR/components.sh"
source "$SCRIPT_DIR/../entity-utils.sh"

PIDFILE="/run/hk/smzjg/watch-components.pid"
# 确保目录存在
mkdir -p "$(dirname "$PIDFILE")"
# 如果旧的 PID 文件存在，先清理
[ -f "$PIDFILE" ] && rm -f "$PIDFILE"
# 写入当前进程 PID
echo $$ > "$PIDFILE"

# fork 到后台并写 PID（核心关键）
echo "🔍 组件守护进程启动..."

# 进入无限循环，定期检查所有组件状态
while true; do
  # 遍历所有组件
  for name in "${components[@]}"; do
  # 获取组件监听的端口号、启动命令、日志文件路径
    port="${component_ports[$name]}"
    cmd="${component_cmds[$name]}"
    log="$COMPONENT_LOG_PATH/${name,,}.log"  # ${name,,} 将组件名转为小写

    # 如果端口未被监听，说明该组件已停止
    if ! ss -tulnp | grep -q ":$port"; then
    # 尝试重启组件，并将结果记录到 WATCH_LOG 中
    watch_and_restart_entity "$name" "$port" "$cmd" "$log"
    fi

  done
  # 每 10 秒检查一次
  sleep 10

done
