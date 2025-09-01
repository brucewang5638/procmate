package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"procmate/pkg/config"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Start 启动指定进程，并根据配置处理其日志输出。
// - 如果配置了 log_file，将使用 lumberjack 进行日志轮转。
// - 否则，日志将被丢弃。
// - 写入 PID 文件。
func Start(proc config.Process) error {
	// 检查进程是否已在运行
	isRunning, _ := IsRunning(proc)
	if isRunning {
		fmt.Printf("⚠️ 进程 '%s' 已在运行。\n", proc.Name)
		return nil
	}

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
	// 默认丢弃所有日志输出，等同于重定向到 /dev/null
	var logWriter io.Writer = io.Discard
	// 获取路径
	logFilePath, err := GetLogFile(proc)
	if err != nil {
		return fmt.Errorf("获取日志文件路径失败: %w", err)
	}

	// 使用 lumberjack 进行日志轮转
	logOptions := config.Cfg.Settings.LogOptions
	logWriter = &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    logOptions.MaxSizeMB,
		MaxBackups: logOptions.MaxBackups,
		MaxAge:     logOptions.MaxAgeDays,
		Compress:   logOptions.Compress,
		LocalTime:  logOptions.LocalTime,
	}

	// 将进程的标准输出和标准错误都重定向到我们配置的 logWriter
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	// === 启动进程 ===
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动命令 '%s' 失败: %w", proc.Name, err)
	}

	// === 保留pid并持久化到文件 ===
	pid := cmd.Process.Pid
	if err := WritePid(proc, pid); err != nil {
		// 如果写入 PID 文件失败，这很严重，需要进行清理
		cmd.Process.Kill() // 确保杀掉我们刚启动的进程，避免产生僵尸进程
		return fmt.Errorf("为进程 '%s' 写入 PID 文件失败: %w", proc.Name, err)
	}

	return nil
}
