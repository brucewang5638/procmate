package cmd

import (
	"fmt"

	"procmate/pkg/config"
	"procmate/pkg/process"

	"github.com/spf13/cobra"
)

// startCmd 定义了 "start" 子命令
var startCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "启动一个或所有进程",
	Long: `启动在配置文件中定义的一个或所有进程。
如果不提供进程名，则会尝试启动所有 'enabled: true' 的进程。`,
	// Args 字段用于校验参数。cobra.MaximumNArgs(1) 表示该命令最多只接受1个参数。
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查用户是否指定了要启动的单个进程名
		if len(args) > 0 {
			// 启动单个进程
			processName := args[0]
			var found *config.Process

			// 在配置中查找该进程
			for _, p := range config.Cfg.Processes {
				if p.Name == processName {
					// 使用 &p 不能直接取循环变量地址，所以先拷贝一份
					temp := p
					found = &temp
					break
				}
			}

			if found == nil {
				return fmt.Errorf("错误: 在配置文件中未找到名为 '%s' 的进程", processName)
			}

			// 调用核心启动逻辑
			return process.Start(*found)
		}

		// 启动所有进程
		fmt.Println("正在启动所有已启用的进程...")
		for _, proc := range config.Cfg.Processes {
			if !proc.Enabled {
				continue
			}
			// 调用核心启动逻辑
			if err := process.Start(proc); err != nil {
				// 如果某个进程启动失败，打印错误并继续尝试启动下一个
				fmt.Printf("启动进程 %s 失败: %v\n", proc.Name, err)
			}
		}
		return nil
	},
}

func init() {
	// 将 startCmd 注册为 rootCmd 的子命令
	rootCmd.AddCommand(startCmd)
}
