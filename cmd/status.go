package cmd

import (
	"fmt"
	"procmate/pkg/config"
	"procmate/pkg/process"

	"github.com/spf13/cobra"
)

// statusCmd 代表 'procmate status' 命令
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "检查并显示所有已定义进程的状态 ⚡",
	Long: `遍历配置文件中定义的所有进程，通过检查其端口来
确定它们是否在线，并以表格形式显示结果。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 打印状态表格的表头
		fmt.Println("NAME\t\tGROUP\t\tPORT\t\tSTATUS")
		fmt.Println("----\t\t-----\t\t----\t\t------")

		// 遍历从配置文件中加载的所有进程
		// config.Cfg 是我们在 root.go 中加载并赋值的全局配置
		for _, proc := range config.Cfg.Processes {
			// 如果进程被禁用 (enabled: false)，则直接跳过，不显示其状态
			if !proc.Enabled {
				continue
			}
			var status string
			isOnline := process.CheckPort(proc.Port)

			if isOnline {
				// ONLINE 用绿色
				status = "\033[32m✅ ONLINE\033[0m"
			} else {
				// OFFLINE 用红色
				status = "\033[31m❌ OFFLINE\033[0m"
			}

			fmt.Printf("%s\t\t%s\t\t%d\t\t%s\n", proc.Name, proc.Group, proc.Port, status)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
