package process

import (
	"fmt"
	"os"
	"strconv"
	"syscall" // 使用操作系统信号
	"time"

	"procmate/pkg/config"
)

// Stop 函数负责停止一个指定的进程。
func Stop(proc config.Process) error {
	fmt.Printf("正在尝试停止进程: %s...\n", proc.Name)

	// 1. 获取 PID 文件路径
	pidFilePath, err := GetPidFile(proc.Name)
	if err != nil {
		return fmt.Errorf("获取PID文件路径失败: %w", err)
	}

	// 2. 检查 PID 文件是否存在
	if _, err := os.Stat(pidFilePath); os.IsNotExist(err) {
		// 如果 PID 文件不存在，我们认为进程已经停止了
		fmt.Printf("✅ 进程 '%s' 已停止 (PID 文件未找到)。\n", proc.Name)
		return nil
	}

	// 3. 读取 PID 文件内容
	content, err := os.ReadFile(pidFilePath)
	if err != nil {
		return fmt.Errorf("读取 PID 文件 %s 失败: %w", pidFilePath, err)
	}

	// 4. 解析 PID
	pid, err := strconv.Atoi(string(content))
	if err != nil {
		return fmt.Errorf("解析 PID '%s' 失败: %w", string(content), err)
	}

	// 5. 查找进程
	// 注意：在 Unix-like 系统上，os.FindProcess 即使进程不存在也不会报错。
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("查找 PID 为 %d 的进程失败: %w", pid, err)
	}

	// 6. 发送 SIGTERM 信号，请求进程优雅退出
	fmt.Printf("⏳ 正在向 PID %d 发送 SIGTERM 信号，请求进程 '%s' 优雅退出...\n",
		pid, proc.Name)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// 如果发送信号失败（比如进程已不存在），打印信息但继续尝试清理
		fmt.Printf("发送 SIGTERM 信号失败: %v。可能进程已退出。\n", err)
	}

	// 7. 等待并验证进程是否已停止
	stopped := false
	for i := 0; i < 10; i++ { // 最多等待10秒
		// 发送一个“空”信号 (signal 0)，用于检查进程是否存在
		if err := process.Signal(syscall.Signal(0)); err != nil {
			stopped = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	// 8. 如果进程仍然存在，发送 SIGKILL 强制终止
	if !stopped {
		fmt.Printf("⚠️ 进程 '%s' (PID: %d) 在10秒内未响应 SIGTERM，正在发送 SIGKILL 强制终止...\n",
			proc.Name, pid)
		if err := process.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("发送 SIGKILL 信号失败: %w", err)
		}
	}

	// 9. 清理 PID 文件
	if err := os.Remove(pidFilePath); err != nil {
		return fmt.Errorf("清理 PID 文件 %s 失败: %w", pidFilePath, err)
	}

	fmt.Printf("✅ 进程 '%s' 已成功停止。\n", proc.Name)
	return nil
}
