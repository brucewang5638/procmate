# Procmate - 进程伴侣

一个使用 Go 语言编写的、强大且可配置的命令行工具，用于管理和守护本地长时间运行的进程。

## ✨ 功能特性

- **统一管理**: 通过一个 `config.yaml` 文件，集中管理所有需要运行的进程。
- **生命周期控制**: 支持 `start`, `stop`, `status`, `restart` 等完整的生命周期命令。
- **后台守护**: `watch` 命令可以作为守护进程，持续监控并自动重启意外崩溃的进程。
- **高度可配置**:

  - 支持自定义启动/停止超时。
  - 支持为进程注入环境变量。
  - 支持自定义日志文件路径。

- **日志记录**: 自动将每个进程的输出重定向到指定的日志文件。
- **Systemd 集成**: 可以轻松地被集成为一个标准的 Linux 系统服务，实现开机自启。
- **高可移植性**: 可被静态编译为单一的二进制文件，在不同版本的 Linux 系统中运行，无 `glibc` 依赖问题。

## 🚀 安装

1. **编译**

   在项目根目录下，运行以下命令来编译一个静态链接的、可在任何 Linux (amd64) 系统上运行的二进制文件：

   ```bash
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o procmate main.go
   ```

2. **放置到系统路径**

   将编译好的 `procmate` 文件移动到系统的可执行路径下，以便在任何地方都能调用它。

   ```bash
    tar -zxf procmate_xxx.tar.gz

    cd  procmate_xxx

    chmod +x install.sh

    ./install.sh

   ```

## ⚙️ 配置

`procmate` 的所有行为都由一个 `config.yaml` 文件控制。您需要在使用前创建这个文件。

**示例 `config.yaml`:**

```yaml
# 全局默认设置
settings:
  runtime_dir: /tmp/procmate # 运行时文件 (pid, logs) 的根目录
  default_start_timeout_sec: 10 # 默认启动超时 (秒)
  default_stop_timeout_sec: 10 # 默认停止超时 (秒)
  watch_interval_sec: 10 # 'watch' 命令的轮询周期 (秒)

processes:
  - name: web-server-1
    group: web
    command: "while true; do { echo -e 'HTTP/1.1 200 OK\\r\\n\\r\\nHello'; } | nc -l -p 8080; done"
    workdir: "/tmp"
    port: 8080
    enabled: true
    start_timeout_sec: 5 # 使用自定义的启动超时

  - name: api-server-2
    group: api
    command: "..."
    workdir: "/app/api"
    port: 8081
    enabled: true
    environment:
      API_KEY: "your-secret-key"
    log_file: "/var/log/my_api_server.log"
```

## 💡 使用方法

确保您的 `config.yaml` 文件存在。您可以通过 `--config` 或 `-f` 标志来指定其路径。

- **检查所有进程的状态**

  ```bash
  procmate status
  ```

- **启动所有已启用的进程**

  ```bash
  procmate start
  # 或者
  procmate start all
  ```

- **启动单个进程**

  ```bash
  procmate start [name]
  ```

- **停止所有进程**

  ```bash
  procmate stop
  ```

- **停止单个进程**

  ```bash
  procmate stop [name]
  ```

- **启动守护模式** (通常在前台运行用于调试，或通过 systemd 在后台运行)

  ```bash
  procmate watch
  ```

- **指定配置文件路径**

  ```bash
  procmate --config /etc/procmate/config.yaml status
  ```

## 🛡️ 作为 Systemd 服务运行

为了实现后台守护和开机自启，建议将 `procmate` 注册为一个 `systemd` 服务。

1. **创建服务文件**

   ```bash
   sudo nano /etc/systemd/system/procmate.service
   ```

2. **填入以下内容** (请务必使用您配置文件的真实绝对路径):

   ```ini
   [Unit]
   Description=Procmate Process Companion Service
   After=network.target

   [Service]
   Type=simple
   ExecStart=/usr/local/bin/procmate watch --config /path/to/your/config.yaml
   User=root
   Restart=on-failure
   RestartSec=5s

   [Install]
   WantedBy=multi-user.target
   ```

3. **管理服务**

   ```bash
   # 重载 systemd 配置
   sudo systemctl daemon-reload

   # 设置开机自启
   sudo systemctl enable procmate.service

   # 立即启动服务
   sudo systemctl start procmate.service

   # 查看服务状态
   sudo systemctl status procmate.service

   # 查看实时日志
   sudo journalctl -u procmate.service -f
   ```
