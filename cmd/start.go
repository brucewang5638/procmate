package cmd

import (
	"fmt"

	"procmate/pkg/config"
	"procmate/pkg/process"

	"github.com/spf13/cobra"
)

// startCmd 定义了 "start" 子命令
var startCmd = &cobra.Command{
	Use:   "start [name|all]",
	Short: "启动一个或所有进程 ⚡",
	Long: `启动在配置文件中定义的一个或所有进程。
如果不提供进程名，则会尝试启动所有 'enabled: true' 的进程。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("⚠️ 请指定要启动的进程名，或者使用 'all' 启动所有进程")
		}

		if args[0] == "all" {
			fmt.Println("⚡ 正在启动所有已启用的进程...")
			for _, proc := range config.Cfg.Processes {
				if !proc.Enabled {
					continue
				}
				if err := process.Start(proc); err != nil {
					fmt.Printf("❌ 启动进程 %s 失败: %v\n", proc.Name, err)
				} else {
					fmt.Printf("✅ 进程 %s 已启动成功\n", proc.Name)
				}
			}
			return nil
		}

		// 启动单个进程
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

		fmt.Printf("⚡ 正在启动进程 %s...\n", found.Name)
		if err := process.Start(*found); err != nil {
			fmt.Printf("❌ 启动进程 %s 失败: %v\n", found.Name, err)
			return err
		}
		fmt.Printf("✅ 进程 %s 已启动成功\n", found.Name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
