#!/bin/bash

# ✅ 基础路径（所有组件根目录）
COMPONENT_BASE_PATH="/opt/hk/component"
COMPONENT_LOG_PATH="/var/log/hk/smzjg/component"

# 定义有哪些组件，执行顺序依此！！
components=("nginx" "redis" "elasticsearch" "nacos" "zookeeper" "kafka")

# ✅ 组件名 → 端口号
declare -A component_ports=(
  ["nginx"]=19704
  ["elasticsearch"]=19702
  ["zookeeper"]=12181
  ["kafka"]=19701
  ["nacos"]=18848
  ["redis"]=19703
  ["mariadb"]=3306
)

# ✅ 组件名 → 启动命令
declare -A component_cmds=(
  ["nginx"]="$COMPONENT_BASE_PATH/hknginx1/nginx -p $COMPONENT_BASE_PATH/hknginx1 -c $COMPONENT_BASE_PATH/hknginx1/nginx.conf"
  ["elasticsearch"]="$COMPONENT_BASE_PATH/elasticsearch-7.6.2/bin/elasticsearch"
  ["zookeeper"]="$COMPONENT_BASE_PATH/kafka/bin/zookeeper-server-start.sh $COMPONENT_BASE_PATH/kafka/config/zookeeper.properties"
  ["kafka"]="$COMPONENT_BASE_PATH/kafka/bin/kafka-server-start.sh $COMPONENT_BASE_PATH/kafka/config/server.properties"
  ["nacos"]="sh $COMPONENT_BASE_PATH/nacos/bin/startup.sh -m standalone"
  ["redis"]="$COMPONENT_BASE_PATH/redis/bin/redis-server $COMPONENT_BASE_PATH/redis/bin/redis.conf"
  ["mariadb"]="systemctl start mariadb.service"
)
