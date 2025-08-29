package process

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"procmate/pkg/config"

	psnet "github.com/shirou/gopsutil/v3/net"

	gops "github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo 包含了一个进程在运行时的所有动态信息。
type ProcessInfo struct {
	Name           string
	IsRunning      bool
	PID            int
	Uptime         time.Duration
	CPUPercent     float64
	MemoryRSS      float64 // 单位: MB
	ListeningPorts []string
}

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

// GetProcessInfo 通过PID文件和系统调用，获取一个进程的详细运行时信息。
// 这是获取进程状态的核心功能，将系统交互的逻辑与命令行展示的逻辑彻底分离。
func GetProcessInfo(proc config.Process) (*ProcessInfo, error) {
	// 初始化返回结构体，默认进程为离线状态
	info := &ProcessInfo{
		Name:      proc.Name,
		IsRunning: false,
	}

	// 1. 从 PID 文件中读取 PID
	pid, err := ReadPid(proc)
	if err != nil || pid == 0 {
		// 读取失败或PID为0，均视为进程不在线。这不是一个错误，而是正常状态。
		return info, nil
	}

	// 2. 使用 gopsutil 验证进程是否真实存在
	p, err := gops.NewProcess(int32(pid))
	if err != nil {
		// PID 文件存在，但系统中已找不到该进程（例如，进程崩溃但PID文件未清理）。
		// 同样视为离线。
		return info, nil
	}

	// --- 如果代码能执行到这里，说明进程确认在线 ---
	info.IsRunning = true
	info.PID = pid

	// 3. 获取进程的详细运行时信息
	// 获取进程启动时间，并计算已运行时间
	if createTime, err := p.CreateTime(); err == nil {
		info.Uptime = time.Since(time.UnixMilli(createTime)).Round(time.Second)
	}

	// 获取 CPU 使用率
	if cpuPercent, err := p.CPUPercent(); err == nil {
		info.CPUPercent = cpuPercent
	}

	// 获取内存使用情况 (RSS, 物理内存)
	if memInfo, err := p.MemoryInfo(); err == nil {
		info.MemoryRSS = float64(memInfo.RSS) / 1024 / 1024 // 字节转换为 MB
	}

	// 获取网络连接，并筛选出正在监听的 TCP 端口
	connections, err := psnet.ConnectionsPid("tcp", int32(pid))
	if err != nil {
		// 不要让这个错误中断整个流程，但必须打印到标准错误流，以便用户诊断。
		// 这通常是一个权限问题，提示用户使用 sudo。
		fmt.Fprintf(os.Stderr, "[警告] 无法获取进程 '%s' (PID: %d) 的网络连接: %v。请尝试使用 sudo 运行。\n", proc.Name, pid, err)
	} else {
		var ports []string
		for _, conn := range connections {
			if conn.Status == "LISTEN" {
				ports = append(ports, fmt.Sprintf("%d", conn.Laddr.Port))
			}
		}
		info.ListeningPorts = ports
	}

	return info, nil
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
