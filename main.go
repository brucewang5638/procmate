package main

import (
	"procmate/cmd"

	"github.com/spf13/cobra"
)

// rootCmd 打包我们应用的基础命令
// 用户没有任何子命令时，该命令会执行
var rootCmd = &cobra.Command{
	Use:   "procmate",
	Short: "一个用于管理服务的命令行工具",
	Long:  "procmate 是一个功能强大的CLI工具，它可以帮助您轻松地启动、停止和监控您的服务。",
}

// 主程序入口
func main() {
	// fmt.Println("procmate, build~~~~")
	cmd.Execute()
}
