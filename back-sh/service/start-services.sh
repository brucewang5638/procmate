#!/bin/bash
set -e

# 获取当前脚本的绝对目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 引用通用变量及脚本
source "$SCRIPT_DIR/services.sh"
source "$SCRIPT_DIR/../entity-utils.sh"

mkdir -p "$SERVICE_LOG_PATH"

echo "🚀 启动所有业务服务..."

# 遍历所有服务，逐一启动
for name in "${services[@]}"; do
  # 获取服务对应的端口号、JAR 包路径、日志文件路径
  # 若未定义，则默认赋值为 0，表示不守护
  PORT="${service_ports[$name]:-0}"
  JAR_PATH="${service_jarpaths[$name]}"
  LOG_DIR="$SERVICE_LOG_PATH/$name" # 每个服务独立目录
  mkdir -p "$LOG_DIR" # 确保目录存在
  LOG_FILE="$LOG_DIR/$name-$(date '+%Y-%m-%d').log" # 日志文件按天分割

  # 如果 JAR 文件存在
  if [ -f "$JAR_PATH" ]; then
    # 构造 Java 启动命令，包含 -Dname 标识（便于 grep 管理）
    CMD="java $JAVA_OPTS -Dname=$name -jar \"$JAR_PATH\""

    # 如果端口号有效（大于 0），表示该服务需要守护
    if [ "$PORT" -gt 0 ]; then
      # 调用统一的启动方法，会检测端口监听状态
      start_entity "$name" "$CMD" "$LOG_FILE" "$PORT"
    else
      # 不做端口监听检测，直接以 nohup 方式启动
      echo "➡️ 启动 $name（不检测端口）"
      eval "nohup $CMD >> \"$LOG_FILE\" 2>&1 &"
    fi

  else
    # 若找不到 JAR 包，则打印警告并跳过
    echo "⚠️ 找不到 JAR 文件：$JAR_PATH，跳过 $name"
  fi

done


echo "✅ 所有服务已启动"
echo "📁 日志目录：$SERVICE_LOG_PATH"

# 👇 最后启动守护进程（后台 + 写 PID，systemd 会读取该 PID）
"$SCRIPT_DIR/watch-services.sh"