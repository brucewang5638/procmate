package cmd

import (
	"fmt"

	"procmate/pkg/config"
	"procmate/pkg/process"

	"github.com/spf13/cobra"
)

// stopCmd 定义了 "stop" 子命令
var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "停止一个或所有正在运行的进程",
	Long: `停止在配置文件中定义的一个或所有进程。
它会通过读取 .pid 文件来找到进程并终止它。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查用户是否指定了要停止的单个进程名
		if len(args) > 0 {
			processName := args[0]
			var found *config.Process

			// 在配置中查找该进程
			for _, p := range config.Cfg.Processes {
				if p.Name == processName {
					// 避免直接取循环变量地址
					temp := p
					found = &temp
					break
				}
			}

			if found == nil {
				return fmt.Errorf("错误: 在配置文件中未找到名为 '%s' 的进程", processName)
			}

			// 调用核心停止逻辑
			return process.Stop(*found)
		}

		// 停止所有进程
		fmt.Println("正在停止所有已启用的进程...")
		// 我们倒序停止，这在有依赖关系的进程中是一种良好的实践
		for i := len(config.Cfg.Processes) - 1; i >= 0; i-- {
			proc := config.Cfg.Processes[i]
			if !proc.Enabled {
				continue
			}
			if err := process.Stop(proc); err != nil {
				// 即使某个进程停止失败，也不中断循环
				fmt.Printf("停止进程 %s 失败: %v\n", proc.Name, err)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
