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
	Short: "停止一个或所有正在运行的进程 ⏹️",
	Long: `停止在配置文件中定义的一个或所有进程。
它会通过读取 .pid 文件来找到进程并终止它。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("⚠️ 请指定要停止的进程名，或者使用 'all' 停止所有进程")
		}

		if args[0] == "all" {
			fmt.Println("⚡ 正在停止所有已启用的进程...")
			for i := len(config.Cfg.Processes) - 1; i >= 0; i-- {
				proc := config.Cfg.Processes[i]
				if !proc.Enabled {
					continue
				}
				if err := process.Stop(proc); err != nil {
					fmt.Printf("❌ 停止进程 %s 失败: %v\n", proc.Name, err)
				} else {
					fmt.Printf("✅ 进程 %s 已成功停止\n", proc.Name)
				}
			}
			return nil
		}

		// 停止单个进程
		processName := args[0]
		var found *config.Process
		for _, p := range config.Cfg.Processes {
			if p.Name == processName {
				temp := p
				found = &temp
				break
			}
		}

		if found == nil {
			return fmt.Errorf("❌ 错误: 在配置文件中未找到名为 '%s' 的进程", processName)
		}

		fmt.Printf("⚡ 正在停止进程 %s...\n", found.Name)
		if err := process.Stop(*found); err != nil {
			fmt.Printf("❌ 停止进程 %s 失败: %v\n", found.Name, err)
			return err
		}
		fmt.Printf("✅ 进程 %s 已成功停止\n", found.Name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
