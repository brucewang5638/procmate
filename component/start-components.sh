#!/bin/bash
set -e

# 获取当前脚本的绝对目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 引用通用变量及脚本
source "$SCRIPT_DIR/components.sh"
source "$SCRIPT_DIR/../entity-utils.sh"

mkdir -p "$COMPONENT_LOG_PATH"
echo "🚀 启动所有组件服务..."

# 组件es需要依赖ulimt的设置
ulimit -n 65535 || echo "⚠️ 设置 ulimit 失败"

# 遍历所有组件，依次启动
for name in "${components[@]}"; do
  # 获取组件监听的端口号、启动命令、日志文件路径
  port="${component_ports[$name]}"
  cmd="${component_cmds[$name]}"
  log="$COMPONENT_LOG_PATH/${name,,}.log"  # ${name,,} 表示将组件名转为小写

  # 启动该组件，并检测端口监听是否成功
  start_entity "$name" "$cmd" "$log" "$port"
done

echo "✅ 所有组件已成功启动"
echo "📁 日志目录：$COMPONENT_LOG_PATH"

# 👇 最后启动守护进程（后台 + 写 PID，systemd 会读取该 PID）
"$SCRIPT_DIR/watch-components.sh"
