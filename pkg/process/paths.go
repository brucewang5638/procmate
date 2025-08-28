package process

import (
	"fmt"
	"os"
	"path/filepath"
)

// RuntimeDir 定义了存放运行时文件（如PID文件和日志）的目录 。
// 我们选择放在 /tmp/procmate  下，这是一个通用的临时目录。
const RuntimeDir = "/tmp/procmate"

// ensureRuntimeDir 函数确保运行时目录存在。
func ensureRuntimeDir() error {
	// os.MkdirAll  会创建所有必需的父目录，如果目录已存在，它不会 返回错误。
	return os.MkdirAll(RuntimeDir, 0755)
}

// GetPidFile 函数返回指定进程的 PID  文件的标准路径。
func GetPidFile(processName string) (string, error) {
	if err := ensureRuntimeDir(); err != nil {
		return "", err
	}
	return filepath.Join(RuntimeDir, fmt.Sprintf(
		"%s.pid", processName)), nil
}

// GetLogFile  函数返回指定进程的日志文件的标准路径。
func GetLogFile(processName string) (string,
	error) {
	if err := ensureRuntimeDir(); err != nil {
		return "", err
	}
	return filepath.Join(RuntimeDir, fmt.Sprintf(
		"%s.log", processName)), nil
}
