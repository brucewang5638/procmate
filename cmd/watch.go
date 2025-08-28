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
		fmt.Println("每10秒检查一次所有已启用进程的状态。")

		// 创建定时器，每10秒触发一次
		ticker := time.NewTicker(10 * time.Second)
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

		isOnline := process.CheckPort(proc.Port)

		if isOnline {
			// 绿色 ✅ 表示状态正常
			fmt.Printf("\033[32m✔️ 进程 '%s' 状态正常 (端口: %d 在线)\033[0m\n", proc.Name, proc.Port)
		} else {
			// 红色 🚨 表示离线警告
			fmt.Printf("\033[31m🚨 警告: 进程 '%s' (端口: %d) 离线！\033[0m\n", proc.Name, proc.Port)
			// 尝试自动重启
			if err := process.Start(proc); err != nil {
				fmt.Printf("\033[31m❌ 自动重启进程 '%s' 失败: %v\033[0m\n", proc.Name, err)
			} else {
				fmt.Printf("\033[33m⚡ 进程 '%s' 已自动重启。\033[0m\n", proc.Name)
			}
		}
	}
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
