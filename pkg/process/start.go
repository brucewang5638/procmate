package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"procmate/pkg/config"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Start 启动指定进程，并根据配置处理其日志输出。
// - 如果配置了 log_file，将使用 lumberjack 进行日志轮转。
// - 否则，日志将被丢弃。
// - 写入 PID 文件。
// - 启动后会阻塞，直到进程“就绪”或超时。
func Start(proc config.Process) error {
	// 检查进程是否已在运行
	isRunning, _ := IsRunning(proc)
	if isRunning {
		// 如果已经在运行，我们还需要检查它是否就绪
		isReady, _ := IsReady(proc)
		if isReady {
			fmt.Printf("🟡 进程 '%s' 已在运行并就绪。\n", proc.Name)
			return nil
		}
		fmt.Printf("🟠 进程 '%s' 已在运行但尚未就绪，将继续等待...\n", proc.Name)
	} else {
		// === 构造命令 ===
		cmd := exec.Command("bash", "-c", proc.Command)
		cmd.Dir = proc.WorkDir

		// 应用环境变量（继承系统环境 + 进程配置）
		if len(proc.Environment) > 0 {
			cmd.Env = os.Environ()
			for key, val := range proc.Environment {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, val))
			}
		}

		// === 配置日志 ===
		var logWriter io.Writer = io.Discard
		logFilePath, err := GetLogFile(proc)
		if err != nil {
			return fmt.Errorf("获取日志文件路径失败: %w", err)
		}
		logOptions := config.Cfg.Settings.LogOptions
		logWriter = &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    logOptions.MaxSizeMB,
			MaxBackups: logOptions.MaxBackups,
			MaxAge:     logOptions.MaxAgeDays,
			Compress:   logOptions.Compress,
			LocalTime:  logOptions.LocalTime,
		}
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter

		// === 启动进程 ===
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("启动命令 '%s' 失败: %w", proc.Name, err)
		}

		// === 保留pid并持久化到文件 ===
		pid := cmd.Process.Pid
		if err := WritePid(proc, pid); err != nil {
			cmd.Process.Kill()
			return fmt.Errorf("为进程 '%s' 写入 PID 文件失败: %w", proc.Name, err)
		}
		fmt.Printf("... 进程 %s 已启动 (PID: %d)，正在等待其就绪...\n", proc.Name, pid)
	}

	// === 等待进程就绪 ===
	if err := waitForReady(proc); err != nil {
		// 停止失败的进程
		if stopErr := Stop(proc); stopErr != nil {
			fmt.Printf("⚠️ 停止超时的进程 '%s' 失败: %v。可能需要手动清理。\n", proc.Name, stopErr)
		}
		return err
	}

	return nil
}

// waitForReady 会在指定超时时间内等待进程就绪。
// - 就绪则返回 nil
// - 超时则返回 error
func waitForReady(proc config.Process) error {
	// 超时时间：优先用进程自身配置，否则用全局配置
	timeout := time.Duration(config.Cfg.Settings.DefaultStartTimeoutSec) * time.Second
	if proc.StartTimeoutSec > 0 {
		timeout = time.Duration(proc.StartTimeoutSec) * time.Second
	}
	if timeout <= 0 {
		timeout = 60 * time.Second // 最小默认超时
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ready, _ := IsReady(proc)
		if ready {
			return nil // 成功！
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 超时
	return fmt.Errorf("进程 '%s' 在 %v 内未能达到就绪状态", proc.Name, timeout)
}
