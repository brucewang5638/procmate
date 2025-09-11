package cmd

import (
	"context"
	"fmt"
	"strings"

	"procmate/pkg/config"
	"procmate/pkg/process"

	"github.com/spf13/cobra"
)

// startCmd 定义了 "start" 子命令
// 支持按依赖关系并行启动进程，显著提升启动效率
var startCmd = &cobra.Command{
	Use:   "start [service1 service2...|all]",
	Short: "并行启动一个或多个进程 ⚡",
	Long: `按依赖关系分层并行启动进程。

同一层内的进程将并行启动，层与层之间串行执行以确保依赖关系。
这种方式可以显著提升启动效率，特别是在有多个独立服务的情况下。`,
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

		// 2. 解析并确定请求启动的服务列表
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

		// 3. 验证是否有进程需要启动
		if len(requestedProcesses) == 0 {
			fmt.Println("🤔 没有指定要启动的进程，或者没有已启用的进程。")
			return nil
		}

		// 4. 获取分层执行计划（支持并行启动）
		executionLayers, err := process.GetExecutionLayers(allEnabledProcesses, requestedProcesses)
		if err != nil {
			return fmt.Errorf("❌ 无法确定启动计划: %w", err)
		}

		// // 5. 显示执行计划概览
		// fmt.Printf("✅ 启动计划已确定，共 %d 层，将并行启动:\n", len(executionLayers))
		// for i, layer := range executionLayers {
		// 	fmt.Printf("　第 %d 层 (%d 个进程): ", i+1, len(layer))
		// 	for j, p := range layer {
		// 		if j > 0 {
		// 			fmt.Print(", ")
		// 		}
		// 		fmt.Print(p.Name)
		// 	}
		// 	fmt.Println()
		// }
		// fmt.Println("---")

		// 6. 使用智能失败处理的并行启动管理器执行启动
		manager := process.NewParallelStartManager(process.GetSmartParallelStartOptions())
		ctx := context.Background()

		layerResults, err := manager.StartProcessesInLayers(executionLayers, ctx)
		if err != nil {
			return fmt.Errorf("❌ 并行启动失败: %w", err)
		}

		// 7. 显示启动结果
		for _, layerResult := range layerResults {
			// 显示失败的进程详情
			for _, result := range layerResult.Results {
				if !result.Success && !result.IsSkipped {
					fmt.Printf("❌ 进程 %s 启动失败: %v\n", result.Process.Name, result.Error)
					// TODO 这儿应该是并行的去停止
					process.Stop(result.Process)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
