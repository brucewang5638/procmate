package process

import (
	"fmt"
	"os"
	"path/filepath"
	"procmate/pkg/config" // 引入 config 包以访问全局配置
	"strconv"
)

// ensureCommonRuntimeDir 确保运行时目录存在，并返回其路径。
// 如果未配置 runtime_dir，则默认使用 /tmp/procmate。
func ensureCommonRuntimeDir() (string, error) {
	runtimeDir := config.Cfg.Settings.RuntimeDir
	if runtimeDir == "" {
		// 防御性编程：用户没设置时给一个默认值
		runtimeDir = "/tmp/procmate"
	}

	// 创建目录（递归创建父目录），如果已存在不会报错
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return "", err
	}
	return runtimeDir, nil
}

// getPidFile 返回指定进程的 PID 文件路径。
// 格式：<runtime_dir>/<proc.Name>.pid
func getPidFile(proc config.Process) (string, error) {
	runtimeDir, err := ensureCommonRuntimeDir()
	if err != nil {
		return "", err
	}

	// 构造 pids 路径
	pidDir := filepath.Join(runtimeDir, "pids")

	// 确保目录存在
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory '%s': %w", pidDir, err)
	}

	return filepath.Join(pidDir, fmt.Sprintf("%s.pid", proc.Name)), nil
}

// GetLogFile 返回指定进程的日志文件路径。
func GetLogFile(proc config.Process) (string, error) {
	// 默认放在 runtime_dir 下
	runtimeDir, err := ensureCommonRuntimeDir()
	if err != nil {
		return "", err
	}

	// 构造 logs/<proc.Name> 路径
	logDir := filepath.Join(runtimeDir, "logs", proc.Name)

	// 确保目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory '%s': %w", logDir, err)
	}

	return filepath.Join(logDir, fmt.Sprintf("%s.log", proc.Name)), nil
}

// SavePid 保存进程 PID 到对应的 .pid 文件。
func WritePid(proc config.Process, pid int) error {
	pidFilePath, err := getPidFile(proc)
	if err != nil {
		// 如果连获取路径都失败了，直接返回错误
		return fmt.Errorf("获取PID文件路径失败: %w", err)
	}
	// 将整数 PID 转换为字符串
	pidString := strconv.Itoa(pid)

	// 使用 os.WriteFile 将 PID 字符串写入文件。
	// 这个函数会自动处理文件的创建、写入和关闭。
	// 如果写入过程中发生任何错误（如权限不足、磁盘已满），它会返回一个 error
	return os.WriteFile(pidFilePath, []byte(pidString), 0644)
}

// ReadPid 读取进程的 PID，如果文件不存在或内容非法，返回错误。
func ReadPid(proc config.Process) (int, error) {
	pidFile, err := getPidFile(proc)
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, fmt.Errorf("无法解析 PID 文件 %s: %v", pidFile, err)
	}
	return pid, nil
}

// RemovePid 删除进程的 PID 文件。
func RemovePid(proc config.Process) error {
	pidFile, err := getPidFile(proc)
	if err != nil {
		return err
	}
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
