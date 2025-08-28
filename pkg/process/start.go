package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"procmate/pkg/config"
)

// Start 函数负责启动一个指定的进程。
func Start(proc config.Process) error {
	fmt.Printf("正在尝试启动进程: %s...\n", proc.Name)

	// 1. 检查进程是否已在运行
	if CheckPort(proc.Port) {
		fmt.Printf("✅ 进程 '%s' 已在运行 (端口 %d 已被监听)。\n", proc.Name, proc.Port)
		return nil
	}

	// 2. 获取日志文件和 PID 文件的路径
	logFilePath, err := GetLogFile(proc.Name)
	if err != nil {
		return fmt.Errorf("获取日志文件路径失败: %w", err)
	}
	pidFilePath, err := GetPidFile(proc.Name)
	if err != nil {
		return fmt.Errorf("获取PID文件路径失败: %w", err)
	}

	// 3. 创建并准备命令
	// 使用 "bash -c" 来执行命令，这允许我们在 command 字段中使用复杂的 shell 语法，如管道和重定向。
	cmd := exec.Command("bash", "-c", proc.Command)
	cmd.Dir = proc.WorkDir // 设置命令的工作目录

	// 4. 重定向标准输出和标准错误到日志文件
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件 %s 失败: %w", logFilePath, err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// 5. 启动命令（非阻塞）
	// cmd.Start() 会在后台启动命令，并且不会等待它执行完成。
	if err := cmd.Start(); err != nil {
		logFile.Close() // 启动失败时也要确保关闭文件句柄
		return fmt.Errorf("启动命令失败: %w", err)
	}

	// 6. 创建并写入 PID 文件
	pid := cmd.Process.Pid
	err = os.WriteFile(pidFilePath, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		logFile.Close()
		// 如果写入PID文件失败，我们应该尝试杀死刚刚启动的进程，以避免产生僵尸进程。
		cmd.Process.Kill()
		return fmt.Errorf("写入 PID 文件 %s 失败: %w", pidFilePath, err)
	}

	// 7. 等待并验证进程是否成功启动
	// 我们给进程最多10秒的时间来监听端口。
	fmt.Printf("⏳ 进程 '%s' 已启动 (PID: %d)，正在等待端口 %d 变为在线状态...\n",
		proc.Name, pid, proc.Port)

	time.Sleep(1 * time.Second) // 启动后先等待1秒，给进程一点初始化时间。

	success := false
	for i := 0; i < 30; i++ { // 最多检查30次 (大约30秒)
		if CheckPort(proc.Port) {
			success = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	logFile.Close() // 确保日志文件句柄被关闭

	if success {
		fmt.Printf("✅ 进程 '%s' 启动成功！\n", proc.Name)
		return nil
	} else {
		return fmt.Errorf("❌ 进程 '%s' 启动后，在30秒内端口 %d 未变为在线状态",
			proc.Name, proc.Port)
	}
}
