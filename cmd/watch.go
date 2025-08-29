package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"procmate/pkg/config"
	"procmate/pkg/process"

	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "启动守护模式，持续监控并自动重启已关闭的进程 🛡️",
	Long: `这是一个长期运行的命令。它会周期性地检查所有已启用进程的状态，
如果发现某个进程离线，则会自动尝试重新启动它。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("✅ procmate 守护模式已启动... (按 Ctrl+C 退出)")

		watchInterval := config.Cfg.Settings.WatchIntervalSec

		fmt.Printf("每 %d 秒检查一次所有已启用进程的状态。\n", watchInterval)

		// 创建定时器，每 watchInterval 秒触发一次
		ticker := time.NewTicker(time.Duration(watchInterval) * time.Second)

		defer ticker.Stop()

		// 捕获退出信号
		quitChannel := make(chan os.Signal, 1)
		signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)

		// 立即执行一次检查
		checkAndRestartProcesses()

		// 主循环
		for {
			select {
			case <-ticker.C:
				fmt.Println("\n⏰ [TICK] 周期性检查开始...")
				checkAndRestartProcesses()
			case <-quitChannel:
				fmt.Println("\n🛑 收到退出信号，正在关闭守护进程...")
				return nil
			}
		}
	},
}

// checkAndRestartProcesses 封装单次检查和重启逻辑
func checkAndRestartProcesses() {
	for _, proc := range config.Cfg.Processes {
		if !proc.Enabled {
			continue
		}

		// 检查是否允许正常
		isRunning := process.IsRunning(proc)
		isReady, err := process.IsReady(proc)
		if err != nil {
			fmt.Printf("❌ 检查进程就绪状态失败: %v\033[0m\n", err)
			return
		}

		if isRunning {
			if isReady {
				// 绿色表示运行且就绪
				fmt.Printf("\033[32m✔️ 进程 '%s' 运行中且就绪\033[0m\n", proc.Name)
			} else {
				// 黄色表示运行但未就绪
				fmt.Printf("\033[33m♻️ 进程 '%s' 运行中，但未就绪\033[0m\n", proc.Name)

				// 检查是否超时，如果超时就kill
				checkProcessTimeoutNoReady(proc)
			}
		} else {
			// 红色 🚨 表示离线警告
			fmt.Printf("\033[31m🚨 警告: 进程 '%s' 离线！\033[0m\n", proc.Name)
			// 尝试自动重启
			if err := process.Start(proc); err != nil {
				fmt.Printf("\033[31m❌ 自动重启进程 '%s' 失败: %v\033[0m\n", proc.Name, err)
			} else {
				fmt.Printf("\033[33m⚡ 进程 '%s' 已自动重启。\033[0m\n", proc.Name)
			}
		}

	}
}

// checkProcessTimeoutNoReady 等待一个已启动的进程达到就绪状态，如果超时则终止它
func checkProcessTimeoutNoReady(proc config.Process) {
	// 计算超时时间
	timeoutSec := proc.StartTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = config.Cfg.Settings.DefaultStartTimeoutSec
	}
	if timeoutSec <= 0 {
		timeoutSec = 60 // 提供一个最终的默认值，防止两者都未配置
	}
	timeoutDuration := time.Duration(timeoutSec) * time.Second

	info, err := process.GetProcessInfo(proc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取进程 '%s' 信息时发生意外错误: %v\n", proc.Name, err)
		return
	}

	// 如果运行时间超过了配置的超时阈值，则认为进程卡住，执行终止操作。
	if info.Uptime > timeoutDuration {
		fmt.Printf("\033[31m🚨 警告: 进程 '%s' 运行已超过 %d 秒但仍未就绪，将被强制终止。\033[0m\n", proc.Name, timeoutDuration)

		if err := process.Stop(proc); err != nil {
			fmt.Printf("\033[31m❌ 终止超时进程 '%s' 失败: %v\033[0m\n", proc.Name, err)
		} else {
			fmt.Printf("\033[33m⚡ 超时进程 '%s' 已成功终止。\033[0m\n", proc.Name)
		}
	}
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
