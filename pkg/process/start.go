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
	if IsRunning(proc) {
		fmt.Printf("✅ 进程 '%s' 已在运行。\n", proc.Name)
		return nil
	}

	runtimeDir := config.Cfg.Settings

	// === 构造命令 ===
	cmd := exec.Command("bash", "-c", proc.Command)
	cmd.Dir = proc.WorkDir

	// === 配置日志 ===
	// 默认丢弃所有日志输出，等同于重定向到 /dev/null
	var logWriter io.Writer = io.Discard

	// === 获取路径 ===
	logFilePath, err := GetLogFile(proc)
	if err != nil {
		return fmt.Errorf("获取日志文件路径失败: %w", err)
	}

	// 如果在 config.yaml 中为该进程配置了 log_file，则使用 lumberjack 进行日志轮转
	logWriter = &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    LogOptions.MaxSizeMB,       // 在轮转之前，日志文件的最大大小（以MB为单位）
		MaxBackups: proc.LogOptions.MaxBackups, // 保留的旧日志文件的最大数量
		MaxAge:     proc.LogOptions.MaxAgeDays, // 保留旧日志文件的最大天数
		Compress:   proc.LogOptions.Compress,   // 是否压缩/归档旧日志文件
		LocalTime:  true,                       // 使用本地时间创建时间戳
	}
	// 将进程的标准输出和标准错误都重定向到我们配置的 logWriter
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	// 应用环境变量（继承系统环境 + 进程配置）
	if len(proc.Environment) > 0 {
		cmd.Env = os.Environ()
		for key, val := range proc.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, val))
		}
	}

	// === 启动进程 ===
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动命令 '%s' 失败: %w", proc.Name, err)
	}

	pid := cmd.Process.Pid
	if err := WritePid(proc, pid); err != nil {
		// 如果写入 PID 文件失败，这很严重，需要进行清理
		cmd.Process.Kill() // 确保杀掉我们刚启动的进程，避免产生僵尸进程
		return fmt.Errorf("为进程 '%s' 写入 PID 文件失败: %w", proc.Name, err)
	}

	fmt.Printf("✅ 进程 '%s' 已成功启动 (PID: %d)。\n", proc.Name, pid)
	return nil
}
