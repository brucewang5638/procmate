组件服务管理脚本的 **使用说明书**，适用于终端用户或运维人员使用时参考。该说明书尽量简洁实用，涵盖了 **组件启动、守护、停止** 各类脚本功能与参数含义。

------

# 📘 组件服务统一管理脚本说明书

本套脚本用于统一管理项目组件及服务（如 Elasticsearch、Nacos 等组件及Framework.jar、Assets.jar等服务），包含启动、守护监控、停止等常用运维操作，适用于 `/opt/hk/$项目`名 下所有组件及服务。

------

### 📌<strong>执行记得加上权限</strong>📌: chmod +x -R /opt/hk/${项目名}/script/

------

## 📈 与旧版启动脚本的对比

统一组件管理脚本的设计，解决了旧版散装启动脚本中的众多痛点，具备更强的可维护性与可观测性。

| 对比项             | 旧版脚本方式                                      | ✅ 统一管理脚本（当前）                               |
| ------------------ | ------------------------------------------------- | ---------------------------------------------------- |
| 🧩 配置集中管理     | 各启动脚本分散，难以统一维护                      | `components.sh / services.sh` 集中配置所有组件和端口 |
| 🧠 启动判断能力     | 无任何判断，需要手动命令方式来自己选择行为        | 自动判断是否已启动，避免重复拉起或冲突               |
| ⏱️ 守护能力         | 没有组件的守护，服务的守护也是一个jar一个极难管理 | 提供 `watch-xxxs.sh` 脚本 + 支持 systemd 持续守护    |
| 📄 日志输出         | 无日志输出（依赖jar本身的日志框架）               | 每个组件独立日志文件，支持时间戳分隔                 |
| 🛑 停止机制         | 无法优雅关闭                                      | 按路径精确匹配进程，优雅终止，支持单个或全部         |
| 🔁 systemd 集成能力 | n个服务n个service                                 | 只需要component、service两个service                  |
| 🧩 扩展新组件       | 每次要写多个新脚本                                | 只需新增配置项即可，无需写脚本                       |
| 🛠️ 问题定位         | 启动失败后无法快速排查原因                        | 每步执行均有日志输出和路径提示                       |

---

### 🔧 **较旧版启动脚本优势**

* ✅ 统一集中式管理，更好维护
  所有组件和服务集中定义于 components.sh / services.sh，只需维护一份配置即可，无需到处找脚本、复制粘贴修改。
* ✅ 批量启动、守护、停止操作更简洁
  旧版脚本一个服务配一个 systemd 单元，数量庞大、耦合性高；而新版仅需两个服务（组件 + 服务），即可覆盖所有资源管理。
* ✅ 自带守护能力，支持 systemd 集成
  提供 watch-xxxs.sh 脚本支持持续运行和端口守护，同时天然支持 systemd，具备自动重启、开机自启、日志归档等特性。
* ✅ 进程管理更精准，避免误杀误拉
  旧脚本用 ps + grep 或 killall，极易误操作；新版通过路径+端口精确匹配，支持优雅终止与强制 kill。
* ✅ 可扩展性强，新增组件无需新写脚本
  仅需在配置文件中新增对应组件项和启动命令，立刻即可纳入统一管理，极大简化扩展成本。
* ✅ 每步操作均带可视化提示，运维体验佳
  启动/停止/守护均带 emoji + 日志提示，失败有详细输出路径，快速定位问题来源。
* ✅ 日志输出结构标准化，便于接入日志平台
  所有组件日志统一输出至 /var/log/hk/${项目名}/ 下，结构清晰、按日期拆分，方便 ELK、Loki 等日志系统自动采集。

------

## 📂 目录结构说明

```
/opt/hk/${项目名}/script/
└── entity-utils.sh              # 通用进程管理工具函数库
├── component                    # 组件管理脚本
├──── components.sh              # 组件配置文件：定义有哪些组件、其端口号与启动命令(最常修改之处！！！)
├──── start-components.sh     # 启动所有组件服务
├──── watch-components.sh     # 守护组件服务（自动重启）
├──── stop-components.sh      # 停止组件服务（支持指定名称）
├── service                      # 服务管理脚本
├──── services.sh                # 服务配置文件：定义有哪些服务、其端口号与jar路径(最常修改之处！！！)
├──── start-components.sh     # 参考组件说明
├──── watch-components.sh     # 参考组件说明
├──── stop-components.sh      # 参考组件说明

```

------

## 🚀 启动脚本

### `./start-xxxs.sh`

用于批量启动所有已配置的组件服务。

- ✅ 自动检测组件是否已启动（通过端口号）
- ✅ 支持后台运行并输出日志

#### 示例：

```bash
./start-components.sh
```

#### 示例输出:

```bash
[root@localhost compoment]# ./start-components.sh 
🚀 启动所有组件服务...
✅ Zookeeper 已在运行，跳过启动
➡️ 启动 Nacos
⏳ 正在等待 Nacos 启动（监听端口 18848）........
✅ Nacos 启动成功
✅ 所有组件已成功启动
📁 日志目录：/var/log/hk/${项目名}/component
```

日志保存在：

```
/var/log/hk/${项目名}/component/<组件名>.log
或者
/var/log/hk/${项目名}/component/<组件名>/<组件名>-YYYY-MM-DD.log
```

------

## 🛡️ 守护脚本

### `./watch-xxxs.sh`

用于持续监控所有组件的运行状态，发现端口关闭后自动重启。

- 默认每 10 秒轮询一次
- ✅ 支持已启动时不重复拉起
- ✅ 支持 systemd 守护方式随系统自动运行

#### 示例：

```bash
./watch-components.sh
```

> ✅ 建议在 systemd 中配置为服务项（如 `hk-components.service`）实现自动守护

------

## 🛑 停止脚本

### `./stop-xxxs.sh [组件名|all]`

用于优雅或强制停止组件进程。

- ✅ 支持按组件名停止单个服务
- ✅ `all` 可停止全部组件
- ✅ 按进程启动路径匹配组件，不误伤其他服务

#### 参数说明：

| 参数   | 示例  | 说明           |
| ------ | ----- | -------------- |
| 组件名 | Kafka | 只停止指定组件 |
| all    | all   | 停止所有组件   |

#### 示例：

```bash
./stop-components.sh Kafka          # 停止 Kafka
./stop-components.sh Redis          # 停止 Redis
./stop-components.sh all            # 停止所有组件
```

#### 示例输出:

```bash
[root@localhost compoment]# ./stop-components.sh nacos1
❌ 无效组件名：nacos1
✅ 用法: ./stop-components.sh [组件名 | all]
📋 可用组件名：
  - Zookeeper
  - Nacos
  - Redis
  - Elasticsearch
  - Kafka
  - MariaDB
  - all   # 停止所有组件
[root@localhost compoment]# ./stop-components.sh Nacos
🛑 正在停止组件进程匹配：/opt/hk/${项目名}/component/(nacos)
⛔ 停止中：/opt/hk/${项目名}/component/(nacos)
⌛ 等待 5 秒终止中...
⚠️ 仍有残留进程，强制关闭...
✅ 停止完成
```



------

## 🧩 `xxxs.sh` 配置说明

组件配置参数：

```bash
# 组件 → 端口号
declare -A components=(
  ["Elasticsearch"]=19702
  ["Kafka"]=19701
  ...
)

# 组件 → 启动命令
declare -A component_cmds=(
  ["Kafka"]="/opt/hk/${项目名}/component/kafka/bin/kafka-server-start.sh ..."
  ...
)
```

> # 💡 **所以我们最常修改的是配置** ！！！
>
> 例如：组件的端口号和启动命令、服务的端口号和jar路径

## 🔧 命令行方式调用（强烈建议）

```sh
ln -sf /opt/hk/smzjg/script/service/hk-smzjg-service.sh /usr/local/bin/hk-smzjg-service
ln -sf /opt/hk/smzjg/script/component/hk-smzjg-component.sh /usr/local/bin/hk-smzjg-component
```

执行示例：

```
[root@192 ~]# hk-smzjg-component status
正在实时查看 [] 的状态... (按 Ctrl+C 退出)
-- Logs begin at Sun 2025-08-24 17:11:04 CST. --
Aug 24 17:11:20 192.168.1.66 systemd[1]: Started SMZJG 所有组件统一启动服务.
Aug 24 17:11:21 192.168.1.66 start-components.sh[907]: 🚀 启动所有组件服务...
Aug 24 17:15:01 192.168.1.66 start-components.sh[907]: ✅ 所有组件已成功启动
Aug 24 17:15:01 192.168.1.66 start-components.sh[907]: 📁 日志目录：/var/log/hk/smzjg/component
Aug 24 17:15:01 192.168.1.66 start-components.sh[907]: 🔍 组件守护进程启动...
```



## 🔧 systemd 启动支持（可选）

你可以将守护脚本注册为开机启动服务：

```bash
sudo systemctl enable hk-smzjg-components.service
sudo systemctl start hk-smzjg-components.service
```

```bash
sudo systemctl enable hk-smzjg-services.service 
sudo systemctl start  hk-smzjg-services.service 
```

服务文件位于：

```
/etc/systemd/system/hk-smzjg-components.service
```

```
/etc/systemd/system/hk-smzjg-services.service 
```

内容参考：

```ini
[Unit]
Description=SMZJG 所有组件统一启动
After=network.target
Wants=network.target

[Service]
User=root
Type=simple

ExecStart=/opt/hk/smzjg/script/component/start-components.sh
ExecStop=/opt/hk/smzjg/script/component/stop-components.sh all

# 设置 Restart 策略用于崩溃时自动拉起
Restart=on-failure
RestartSec=5s

# 日志输出
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

```ini
[Unit]
Description=SMZJG 所有服务统一启动
After=network.target hk-smzjg-components.service
Requires=hk-smzjg-components.service

[Service]
User=root
Type=simple
# 守护脚本用于监控并自动重启组件
ExecStart=/opt/hk/smzjg/script/service/start-services.sh
ExecStop=/opt/hk/smzjg/script/service/stop-services.sh all

# 设置 Restart 策略用于崩溃时自动拉起
Restart=on-failure
RestartSec=5s

# 日志输出
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```



日志查看命令: 

```sh
journalctl -u ${服务名} -f

例如： journalctl -u hk-smzjg-components.service -f
```

------



## 📌 版本roadmap

| 版本号 | 更新内容                                |
| :----- | --------------------------------------- |
| V4     | 支持命令行工具式调用(借助软链方式)      |
| V3     | 解决与systemd的绑定问题、执行顺序可定制 |
| V2     | 抽象出通用进程管理工具函数库            |
| V1     | -                                       |

## 📌 常见问题

| 问题描述       | 解决方案                                               |
| -------------- | ------------------------------------------------------ |
| 日志在哪？     | 默认保存在 `/var/log/hk/${项目名}/component/<组件名>/` |
| 启动失败？     | 会有日志路径提示，查看该日志即可                       |
| 守护没生效？   | 确保 `watch-components.sh` 在运行或通过 systemd 管理   |
| 组件没被停止？ | 检查正则是否匹配到正确路径，或增加日志打印调试         |

------

