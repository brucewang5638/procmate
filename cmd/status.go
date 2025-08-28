package cmd

import (
	"fmt"
	"procmate/pkg/config"  // 导入 config 包以访问全局配置
	"procmate/pkg/process" // 导入 process 包以使用我们即将创建的检查逻辑

	"github.com/spf13/cobra"
)

// statusCmd 代表 'procmate status' 命令
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "检查并显示所有已定义进程的状态",
	Long: `遍历配置文件中定义的所有进程，通过检查其端口来
确定它们是否在线，并以表格形式显示结果。`,
	// RunE 是一个带有错误返回的 Run 函数。如果它返回一个 error，Cobra 会打印出这个错误。
	RunE: func(cmd *cobra.Command, args []string) error {
		// 打印状态表格的表头
		// 使用制表符 \t 来对齐列，使其更美观
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
			// 调用 process 包中的 CheckPort 函数来检查端口状态
			// (我们将在下一步创建这个函数)
			isOnline := process.CheckPort(proc.Port)

			if isOnline {
				// \033[32m 是 ANSI 转义码，用于将文本颜色变为绿色
				// \033[0m 是重置颜色的代码
				status = "\033[32mONLINE\033[0m"
			} else {
				// \033[31m 是将文本变为红色
				status = "\033[31mOFFLINE\033[0m"
			}

			// 打印格式化后的一行状态信息
			fmt.Printf("%s\t\t%s\t\t%d\t\t%s\n", proc.Name, proc.Group, proc.Port, status)
		}

		return nil // 命令成功执行，返回 nil
	},
}

// 在 init 函数中，将 statusCmd 添加为 rootCmd 的一个子命令。
// 这样 cobra 就知道了 'status' 是 'procmate' 下的一个合法命令。
func init() {
	rootCmd.AddCommand(statusCmd)
}
