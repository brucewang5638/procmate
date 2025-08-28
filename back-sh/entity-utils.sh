#!/bin/bash
# 通用实体启动/停止/监控工具库
# 可用于组件（components）或服务（services）脚本中

# 🧱 启动并等待端口监听
start_entity() {
  name="$1"
  cmd="$2"
  log="$3"
  port="$4"
  max_wait="${5:-90}"

  if ss -tulnp | grep -q ":$port"; then
    echo "✅ $name 已在运行，跳过启动"
    return 0
  fi

  echo "➡️ 启动 $name"
  eval "nohup $cmd >> \"$log\" 2>&1 &"

  echo "⏳ 正在等待 $name 启动（监听端口 $port）"
  for ((i=1; i<=max_wait; i++)); do
    sleep 1
    if ss -tulnp | grep -q ":$port"; then
      echo ""
      echo "✅ $name 启动成功"
      return 0
    fi
      echo -n "." && printf "" >&1
  done

  echo ""
  echo "❌ $name 启动失败，请检查日志 $log"
  return 1
}

# 🛑 停止所有匹配的实体进程
stop_entities_by_pattern() {
  pattern="$1" # 匹配 JAR 路径或命令路径
  wait_second="${2:-10}" # 可选参数：默认等待秒数，默认 10 秒

  echo "⛔ 停止中：$pattern"

  pids=$(ps -ef | grep -E "$pattern" | grep -v grep | awk '{print $2}')
  if [ -n "$pids" ]; then
    echo "$pids" | xargs -r kill
    echo "⌛ 正在等待进程退出"

    # 等待进程优雅退出（带点动画）
    for ((i = 1; i <= wait_second; i++)); do
      sleep 1
      still_alive=$(ps -ef | grep -E "$pattern" | grep -v grep | awk '{print $2}')
      if [ -z "$still_alive" ]; then
        echo ""
        echo "✅ 已成功停止"
        return 0
      fi
        echo -n "." && printf "" >&1
    done
  fi

  residual_pids=$(ps -ef | grep -E "$pattern" | grep -v grep | awk '{print $2}')
  if [ -n "$residual_pids" ]; then
    echo "⚠️ 仍有残留进程，强制关闭..."
    echo "$residual_pids" | xargs -r kill -9
  fi

  echo "✅ 停止完成"
}

# 🔁 守护某个实体（重启逻辑）
watch_and_restart_entity() {
  name="$1"
  port="$2"
  cmd="$3"
  log="$4"
  max_wait="${5:-90}"

    # ✅ 参数校验
  if [ -z "$name" ] || [ -z "$port" ] || [ -z "$cmd" ] || [ -z "$log" ]; then
    echo "[$(date '+%F %T')] ❌ 参数缺失，无法守护实体：name=$name, port=$port, cmd=$cmd, log=$log"
    return 1
  fi
  
  echo "[$(date '+%F %T')] ⚠️ $name 端口 $port 未监听，尝试重启..."
  eval "nohup $cmd >> \"$log\" 2>&1 &"

  echo "[$(date '+%F %T')] ⏳ 等待 $name 重启监听中..."
  for ((i=1; i<=max_wait; i++)); do
    sleep 1
    if ss -tulnp | grep -q ":$port"; then
      echo "\n[$(date '+%F %T')] ✅ $name 重启成功"
      return 0
    fi
      echo -n "." && printf "" >&1
  done

  echo "\n[$(date '+%F %T')] ❌ $name 重启失败，请检查日志 $log"
}
