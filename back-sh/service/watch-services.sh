#!/bin/bash
set -e

# 获取当前脚本的绝对目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 引用通用变量及脚本
source "$SCRIPT_DIR/services.sh"
source "$SCRIPT_DIR/../entity-utils.sh"
PIDFILE="/run/hk/smzjg/watch-services.pid"
# 确保目录存在
mkdir -p "$(dirname "$PIDFILE")"
# 如果旧的 PID 文件存在，先清理
[ -f "$PIDFILE" ] && rm -f "$PIDFILE"
# 写入当前进程 PID
echo $$ > "$PIDFILE"

# fork 到后台并写 PID（核心关键）
echo "🔍 业务服务守护进程启动..."

# 无限循环，定期检查所有需要监听端口的服务是否仍存活
while true; do

  # 遍历所有配置了端口监听的服务（即 service_ports 映射）
  for name in "${services[@]}"; do
    # 获取服务对应的端口号、JAR 包路径、日志文件路径
    port="${service_ports[$name]}"
    JAR_PATH="${service_jarpaths[$name]}"
    LOG_FILE="$SERVICE_LOG_PATH/$name/$name-$(date '+%Y-%m-%d').log" # 每个服务独立目录按天分割

    # 构建服务的启动命令，添加 -Dname 参数用于标识服务名
    CMD="java $JAVA_OPTS -Dname=$name -jar \"$JAR_PATH\""

    # 检查该服务的端口是否正在监听
    if ! ss -tulnp | grep -q ":$port"; then
      # 如果端口未监听，说明服务已停止或异常
      watch_and_restart_entity "$name" "$port" "$CMD" "$LOG_FILE"
    fi

  done

  # 每隔 10 秒执行一次检查
  sleep 10

done
