package process

import (
	"errors"
	"fmt"
	"os"
	"procmate/pkg/config"
	"syscall"
	"time"
)

var ErrPidfileNotFound = errors.New("pidfile not found")

// Stop 负责停止一个指定的进程。
func Stop(proc config.Process) error {
	// fmt.Printf("正在尝试停止进程: %s...\n", proc.Name)

	// ===> 使用 ReadPid 辅助函数 <===
	pid, err := ReadPid(proc)
	if err != nil {
		if errors.Is(err, ErrPidfileNotFound) {
			// PID 文件不存在，说明进程已退出，视为成功。
			fmt.Printf("✅ 进程 '%s' 已停止 (PID 文件未找到)。\n", proc.Name)
			return nil
		}
		return err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("查找 PID=%d 的进程失败: %w", pid, err)
	}

	fmt.Printf("⏳ 向 PID=%d 发送 SIGTERM，请求进程 '%s' 优雅退出...\n", pid, proc.Name)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("发送 SIGTERM 失败: %v，可能进程已退出。\n", err)
	}

	// ===> 应用可配置的停止超时 <===
	timeout := config.Cfg.Settings.DefaultStopTimeoutSec
	if proc.StopTimeoutSec > 0 {
		timeout = proc.StopTimeoutSec
	}

	stopped := false
	for i := 0; i < timeout; i++ {
		if err := process.Signal(syscall.Signal(0)); err != nil {
			stopped = true
			break
		}
		time.Sleep(time.Second)
	}

	// 如果进程仍然存在，发送 SIGKILL 强制终止
	if !stopped {
		fmt.Printf("⚠️ 进程 '%s' (PID=%d) 在 %d 秒内未退出，发送 SIGKILL...\n",
			proc.Name, pid, timeout)
		if err := process.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("发送 SIGKILL 失败: %w", err)
		}
	}

	// ===> 使用 RemovePid 辅助函数清理 PID 文件 <===
	if err := RemovePid(proc); err != nil {
		return fmt.Errorf("清理 PID 文件失败: %w", err)
	}

	// 保持沉默由cmd发声
	// fmt.Printf("✅ 进程 '%s' 已成功停止。\n", proc.Name)
	return nil
}
