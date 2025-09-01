package cmd

import (
	"fmt"

	"procmate/pkg/config"
	"procmate/pkg/process"

	"github.com/spf13/cobra"
)

// startCmd 定义了 "start" 子命令
var startCmd = &cobra.Command{
	Use:   "start [service1 service2...|all]",
	Short: "启动一个或多个进程 ⚡",
	Long:  `启动在配置文件中定义的一个或所有进程。如果不提供进程名，则会尝试启动所有 'enabled: true' 的进程。`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. 将所有进程放入 map 以便快速查找
		allProcessesMap := make(map[string]config.Process)
		for _, p := range config.Cfg.Processes {
			allProcessesMap[p.Name] = p
		}

		// 2. 确定请求启动的服务列表
		var requestedServices []string
		if len(args) > 0 && args[0] == "all" {
			fmt.Println("⚡ 计算所有已启用进程的启动顺序...")
			for _, p := range config.Cfg.Processes {
				if p.Enabled {
					requestedServices = append(requestedServices, p.Name)
				}
			}
		} else {
			fmt.Printf("⚡ 计算 %v 的启动顺序...\n", args)
			requestedServices = args
		}

		if len(requestedServices) == 0 {
			fmt.Println("🤔 没有指定要启动的进程，或者没有已启用的进程。")
			return nil
		}

		// 3. 获取包含依赖关系的执行计划
		executionPlan, err := process.GetExecutionPlan(allProcessesMap, requestedServices)
		if err != nil {
			return fmt.Errorf("❌ 无法确定启动计划: %w", err)
		}

		fmt.Println("✅ 启动计划已确定，将按以下顺序启动:")
		for i, p := range executionPlan {
			fmt.Printf("  %d. %s\n", i+1, p.Name)
		}
		fmt.Println("---")

		// 4. 按计划顺序启动进程
		for _, proc := range executionPlan {
			// 检查进程是否已在运行
			isRunning, _ := process.IsRunning(proc)
			if isRunning {
				fmt.Printf("🟡 进程 %s 已在运行，跳过启动。\n", proc.Name)
				continue
			}

			fmt.Printf("⚡ 正在启动进程 %s...\n", proc.Name)
			if err := process.Start(proc); err != nil {
				fmt.Printf("❌ 启动进程 %s 失败: %v\n", proc.Name, err)
				// 决定是否要因为一个失败而停止整个流程
				// 目前我们选择继续尝试启动其他进程
			} else {
				fmt.Printf("✅ 进程 %s 已成功启动\n", proc.Name)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
