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
	Short: "启动守护模式，持续监控并自动重启已关闭的进程",
	Long: `这是一个长期运行的命令。它会周期性地检查所有已启用进程的状态，
如果发现某个进程离线，则会自动尝试重新启动它。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("✅ procmate 守护模式已启动... (按 Ctrl+C 退出)")
		fmt.Println("每10秒检查一次所有已启用进程的状态。")

		// 1. 创建一个定时器，每10秒触发一次
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop() // 确保在函数退出时停止定时器，释放资源

		// 2. 设置一个用于捕获退出信号的通道
		quitChannel := make(chan os.Signal, 1)
		signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)

		// 3. 立即执行一次检查，无需等待第一个10秒
		checkAndRestartProcesses()

		// 4. 启动主循环
		for {
			select {
			case <-ticker.C:
				// 定时器触发
				fmt.Println("\n[TICK] 正在执行周期性检查...")
				checkAndRestartProcesses()
			case <-quitChannel:
				// 收到退出信号
				fmt.Println("\n收到退出信号，正在关闭守护进程...")
				return nil // 优雅退出
			}
		}
	},
}

// checkAndRestartProcesses 封装了单次检查和重启的逻辑
func checkAndRestartProcesses() {
	for _, proc := range config.Cfg.Processes {
		if !proc.Enabled {
			continue
		}

		// 复用 CheckPort 逻辑
		isOnline := process.CheckPort(proc.Port)

		if !isOnline {
			fmt.Printf("🚨 警告: 进程 '%s' (端口: %d) 似乎已离线！\n", proc.Name, proc.Port)
			// 自动重启
			if err := process.Start(proc); err != nil {
				fmt.Printf("❌ 自动重启进程 '%s' 失败: %v\n", proc.Name, err)
			}
		} else {
			fmt.Printf("✔️ 进程 '%s' 状态正常 (端口: %d 在线)。\n", proc.Name, proc.Port)
		}
	}
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
