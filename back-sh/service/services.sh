#!/bin/bash

# ✅ 基础路径（所有服务 JAR 所在根目录）
SERVICE_BASE_PATH="/opt/hk/hkservice"
SERVICE_LOG_PATH="/var/log/hk/smzjg/service"

# ✅ Java 启动参数（可以根据服务重写）
JAVA_OPTS="-Xmx512m"

# 定义有哪些服务，执行顺序依此！！
services=(
  "smzjg-framework"
  "smzjg-assets"
  "smzjg-dispose"
  "smzjg-icpconsole"
  "smzjg-milkyway-console"
  "smzjg-milkyway-service"
  "smzjg-report"
  "smzjg-order"
  "smzjg-smjcq-api"
  "smzjg-smjcq-manage"
  "smzjg-screen"
)

# ✅ 服务名 → JAR 路径映射
declare -A service_jarpaths=(
  ["smzjg-assets"]="$SERVICE_BASE_PATH/assets/bus/bin/hkinfo-asset.jar"
  ["smzjg-framework"]="$SERVICE_BASE_PATH/framework/bus/bin/hkinfo-system.jar"
  ["smzjg-dispose"]="$SERVICE_BASE_PATH/dispose/bus/bin/hkinfo-dispose.jar"
  ["smzjg-icpconsole"]="$SERVICE_BASE_PATH/icp/bus/bin/icp-console.jar"
  ["smzjg-milkyway-console"]="$SERVICE_BASE_PATH/milkyway/bus/console/bin/milkyway.jar"
  ["smzjg-milkyway-service"]="$SERVICE_BASE_PATH/milkyway/bus/service/bin/datasource.jar"
  ["smzjg-report"]="$SERVICE_BASE_PATH/report/bus/bin/hkinfo-zjg-report.jar"
  ["smzjg-screen"]="$SERVICE_BASE_PATH/screen/bus/bin/screen.jar"
  ["smzjg-smjcq-api"]="$SERVICE_BASE_PATH/smjcq/api/bus/bin/hkinfo-smjcq-api.jar"
  ["smzjg-smjcq-manage"]="$SERVICE_BASE_PATH/smjcq/manage/bus/bin/hkinfo-smjcq-manage.jar"
  ["smzjg-order"]="$SERVICE_BASE_PATH/order_tzb/bus/bin/hkinfo-order.jar"
)

# ✅ 可选：服务 → 监听端口（用于 watch）
declare -A service_ports=(
  ["smzjg-assets"]=19715
  ["smzjg-framework"]=19700
  ["smzjg-dispose"]=19712
  ["smzjg-icpconsole"]=19710
  ["smzjg-milkyway-console"]=19705
  ["smzjg-milkyway-service"]=19707
  ["smzjg-report"]=19730
  ["smzjg-screen"]=19713
  ["smzjg-smjcq-api"]=19762
  ["smzjg-smjcq-manage"]=19760
  ["smzjg-order"]=19723
)
