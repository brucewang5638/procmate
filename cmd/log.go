package cmd

import (
	"fmt"

	"procmate/pkg/config" // 导入配置包
	"procmate/pkg/process"

	"github.com/spf13/cobra"
)

// logCmd 定义 log 子命令，用于追踪指定进程当天日志
var logCmd = &cobra.Command{
	Use:   "log [process-name]",
	Short: "追踪指定进程当天的日志 📃",
	Long:  "追踪指定进程当天的日志输出，类似 tail -f。",
	Args:  cobra.ExactArgs(1), // log 命令总是需要一个明确的目标
	RunE: func(cmd *cobra.Command, args []string) error {
		processName := args[0]

		// === 查找进程对象 ===
		var found *config.Process
		for _, p := range config.Cfg.Processes {
			if p.Name == processName {
				temp := p
				found = &temp
				break
			}
		}

		if found == nil {
			// 返回错误，由 Cobra 的调用者处理打印和退出
			return fmt.Errorf("❌ 错误: 在配置文件中未找到名为 '%s' 的进程", processName)
		}

		// === 调用 process 包的 TailLog 逻辑 ===
		if err := process.TailLog(*found); err != nil {
			return fmt.Errorf("❌ 追踪进程 %s 的日志失败: %w", found.Name, err)
		}

		return nil
	},
}

// 将 logCmd 注册到 rootCmd
func init() {
	rootCmd.AddCommand(logCmd)
}
