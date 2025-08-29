package process

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"procmate/pkg/config"
)

// IsRunning 检查指定进程是否由 Procmate 管理且当前正在运行。
// 通过读取 PID 文件获取 PID，然后向该进程发送 Signal 0 验证其存在性。
func IsRunning(proc config.Process) bool {
	// 尝试读取 PID 文件
	pid, err := ReadPid(proc)
	if err != nil {
		// 如果读取失败（文件不存在或内容损坏），认为进程未运行
		return false
	}

	// 查找操作系统中的进程
	// 注意：在 Unix-like 系统上 os.FindProcess 总会成功，即使进程不存在
	process, err := os.FindProcess(pid)
	if err != nil {
		// 理论上非 Windows 系统几乎不会出错
		return false
	}

	// --- 核心技巧 ---
	// Signal 0 不会实际发送信号，只会检查进程是否存在以及是否有权限
	err = process.Signal(syscall.Signal(0))

	// err == nil 表示进程存在且可用
	return err == nil
}

// CheckPort 检查指定 TCP 端口是否被占用。
// 返回 true 表示端口已被占用，false 表示端口空闲。
func CheckPort(port int) bool {
	// --- 增加 0 的判断 ---
	if port == 0 {
		// 0 被视为跳过检查
		return true
	}

	// --- 增加对无效端口的判断 ---
	if port < 0 || port > 65535 {
		// 0 或无效端口被视为空闲
		return false
	}

	// 尝试在本地建立监听
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		// Listen 返回错误通常意味着端口已被占用 (EADDRINUSE)
		return true
	}

	// 监听成功，说明端口空闲，立即关闭监听器
	listener.Close()
	return false
}
