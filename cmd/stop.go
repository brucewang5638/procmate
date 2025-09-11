package cmd

import (
	"context"
	"fmt"
	"strings"

	"procmate/pkg/config"
	"procmate/pkg/process"

	"github.com/spf13/cobra"
)

// stopCmd 定义了 "stop" 子命令
// 支持按依赖关系并行停止进程，显著提升停止效率
var stopCmd = &cobra.Command{
	Use:   "stop [service1 service2...|all]",
	Short: "并行停止一个或多个进程 ⏹️",
	Long: `按依赖关系分层并行停止进程。

从依赖关系的顶层开始停止，层与层之间串行执行以确保依赖关系。
同一层内的进程将并行停止，这种方式可以显著提升停止效率。`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. 构建进程映射表，便于快速查找和验证
		var allEnabledProcesses []config.Process                  // 用于传递给函数
		allEnabledProcessesMap := make(map[string]config.Process) // 用于快速查找和验证
		for _, p := range config.Cfg.Processes {
			if p.Enabled {
				allEnabledProcesses = append(allEnabledProcesses, p)
				allEnabledProcessesMap[p.Name] = p
			}
		}

		// 2. 解析并确定请求停止的服务列表
		var requestedProcesses []config.Process
		if len(args) > 0 {
			if args[0] == "all" {
				requestedProcesses = allEnabledProcesses
			} else {
				var invalidNames []string
				for _, name := range args {
					// 使用 "comma-ok" 语法进行存在性检查
					if process, ok := allEnabledProcessesMap[name]; ok {
						requestedProcesses = append(requestedProcesses, process)
					} else {
						invalidNames = append(invalidNames, name)
					}
				}

				if len(invalidNames) > 0 {
					fmt.Printf("⚠️ 警告：以下服务名称无效或未启用: %s", strings.Join(invalidNames, ", "))
				}
			}
		}

		// 3. 验证是否有进程需要停止
		if len(requestedProcesses) == 0 {
			fmt.Println("🤔 没有指定要停止的进程，或者没有已启用的进程。")
			return nil
		}

		// 4. 获取分层执行计划（支持并行停止）
		executionLayers, err := process.GetExecutionLayers(allEnabledProcesses, requestedProcesses)
		if err != nil {
			return fmt.Errorf("❌ 无法确定停止计划: %w", err)
		}

		// // 5. 显示执行计划概览
		// fmt.Printf("✅ 停止计划已确定，共 %d 层，将并行停止:\n", len(executionLayers))
		// for i := len(executionLayers) - 1; i >= 0; i-- {
		// 	layerIndex := len(executionLayers) - 1 - i
		// 	layer := executionLayers[i]
		// 	fmt.Printf("　第 %d 层 (%d 个进程): ", layerIndex+1, len(layer))
		// 	for j, p := range layer {
		// 		if j > 0 {
		// 			fmt.Print(", ")
		// 		}
		// 		fmt.Print(p.Name)
		// 	}
		// 	fmt.Println()
		// }
		// fmt.Println("---")

		// 6. 使用并行停止管理器执行停止
		manager := process.NewParallelStopManager(process.GetDefaultParallelStopOptions())
		ctx := context.Background()

		layerResults, err := manager.StopProcessesInLayers(executionLayers, ctx)
		if err != nil {
			return fmt.Errorf("❌ 并行停止失败: %w", err)
		}

		for _, layerResult := range layerResults {
			// 显示失败的进程详情
			for _, result := range layerResult.Results {
				if !result.Success && result.WasRunning {
					fmt.Printf("❌ 进程 %s 停止失败: %v\n", result.Process.Name, result.Error)
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
