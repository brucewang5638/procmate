package process

import (
	"fmt"
	"os"
	"os/exec"
	"procmate/pkg/config"
	"time"
)

// Start 启动指定进程，并在配置的超时时间内等待其端口可用。
// - 会写入 PID 文件
// - 会将日志输出重定向到对应的日志文件
// - 支持进程级的环境变量和超时设置
func Start(proc config.Process) error {
	// fmt.Printf("🚀 正在尝试启动进程: %s...\n", proc.Name)

	// 检查是否启动了进程，运行
	if IsRunning(proc) {
		fmt.Printf("✅ 进程 '%s' 已在运行。\n", proc.Name)
		fmt.Printf("✅ 进程 '%s' 占用了端口：\n", 0)
		return nil
	}

	// === 构造命令 ===
	cmd := exec.Command("bash", "-c", proc.Command)
	cmd.Dir = proc.WorkDir

	// 重定向标准输出和标准错误到管道
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取 stdout pipe 失败: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取 stderr pipe 失败: %w", err)
	}

	// 应用环境变量（继承系统环境 + 进程配置）
	if len(proc.Environment) > 0 {
		cmd.Env = os.Environ()
		for key, val := range proc.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, val))
		}
	}

	// === 启动进程 ===
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动命令失败: %w", err)
	}

	// === 后台记录日志 ===
	go handleLogStream(proc, "stdout", stdoutPipe)
	go handleLogStream(proc, "stderr", stderrPipe)

	pid := cmd.Process.Pid
	if err := WritePid(proc, pid); err != nil {
		// 因为如果 WritePid 失败了，需要执行清理操作。
		cmd.Process.Kill()
		return fmt.Errorf("写入 PID 文件失败: %w", err)
	}

	// === 等待端口上线 ===
	timeout := config.Cfg.Settings.DefaultStartTimeoutSec
	// ⚠️ 这儿是将 config.Process.StartTimeoutSec 定义成了 int而不是*int
	// 这样虽然无法精准处理0/未定义，但足够简洁
	if proc.StartTimeoutSec > 0 {
		timeout = proc.StartTimeoutSec
	}

	fmt.Printf("⏳ 进程 '%s' 已启动 (PID: %d)，等待端口 %d 可用 (超时: %d 秒)...\n",
		proc.Name, pid, proc.Port, timeout)

	success := false
	for i := 0; i < timeout; i++ {
		if CheckPort(proc.Port) {
			success = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if success {
		// 保持沉默由cmd发声
		// fmt.Printf("✅ 进程 '%s' 启动成功！\n", proc.Name)
		return nil
	}

	// 启动失败清理 PID 文件
	RemovePid(proc)
	return fmt.Errorf("❌ 进程 '%s' 启动后，在 %d 秒内端口 %d 未变为可用",
		proc.Name, timeout, proc.Port)
}
