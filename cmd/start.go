package cmd

import (
	"context"
	"fmt"

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
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. 构建进程映射表，便于快速查找和验证
		allProcessesMap := make(map[string]config.Process)
		for _, p := range config.Cfg.Processes {
			allProcessesMap[p.Name] = p
		}

		// 2. 解析并确定请求启动的服务列表
		var requestedServices []string
		if len(args) > 0 && args[0] == "all" {
			fmt.Println("⚡ 计算所有已启用进程的并行启动计划...")
			// 收集所有启用的进程
			for _, p := range config.Cfg.Processes {
				if p.Enabled {
					requestedServices = append(requestedServices, p.Name)
				}
			}
		} else {
			fmt.Printf("⚡ 计算 %v 的并行启动计划...\n", args)
			requestedServices = args
		}

		// 3. 验证是否有进程需要启动
		if len(requestedServices) == 0 {
			fmt.Println("🤔 没有指定要启动的进程，或者没有已启用的进程。")
			return nil
		}

		// 4. 获取分层执行计划（支持并行启动）
		executionLayers, err := process.GetExecutionLayers(allProcessesMap, requestedServices)
		if err != nil {
			return fmt.Errorf("❌ 无法确定启动计划: %w", err)
		}

		// 5. 显示执行计划概览
		fmt.Printf("✅ 启动计划已确定，共 %d 层，将并行启动:\n", len(executionLayers))
		for i, layer := range executionLayers {
			fmt.Printf("　第 %d 层 (%d 个进程): ", i+1, len(layer))
			for j, p := range layer {
				if j > 0 {
					fmt.Print(", ")
				}
				fmt.Print(p.Name)
			}
			fmt.Println()
		}
		fmt.Println("---")

		// 6. 使用智能失败处理的并行启动管理器执行启动
		manager := process.NewParallelStartManager(process.GetSmartParallelStartOptions())
		ctx := context.Background()
		
		layerResults, err := manager.StartProcessesInLayers(executionLayers, ctx)
		if err != nil {
			return fmt.Errorf("❌ 并行启动失败: %w", err)
		}

		// 7. 显示启动结果汇总
		fmt.Println()
		fmt.Println("📊 启动结果汇总:")
		totalSuccess := 0
		totalFailure := 0
		totalSkipped := 0
		
		for _, layerResult := range layerResults {
			totalSuccess += layerResult.SuccessCount
			totalFailure += layerResult.FailureCount
			totalSkipped += layerResult.SkippedCount
			
			// 显示失败的进程详情
			for _, result := range layerResult.Results {
				if !result.Success && !result.IsSkipped {
					fmt.Printf("❌ 进程 %s 启动失败: %v\n", result.Process.Name, result.Error)
				}
			}
		}
		
		fmt.Printf("✅ 成功: %d个　❌ 失败: %d个　🟡 跳过: %d个\n", totalSuccess, totalFailure, totalSkipped)
		
		// 8. 根据结果决定命令执行状态
		if totalFailure > 0 {
			// 收集启动失败的进程和成功启动的进程
			var failedProcesses []config.Process
			var successfulProcesses []config.Process
			
			for _, layerResult := range layerResults {
				for _, result := range layerResult.Results {
					if !result.Success && !result.IsSkipped {
						failedProcesses = append(failedProcesses, result.Process)
					} else if result.Success && !result.IsSkipped {
						successfulProcesses = append(successfulProcesses, result.Process)
					}
				}
			}

			// 自动清理启动失败的进程
			if len(failedProcesses) > 0 {
				fmt.Printf("\n🧹 发现 %d 个启动失败的进程，正在自动清理...\n", len(failedProcesses))
				
				fmt.Printf("📋 清理计划：将清理 %d 个启动失败的进程\n", len(failedProcesses))
				for _, proc := range failedProcesses {
					fmt.Printf("　- %s\n", proc.Name)
				}
				
				// 对于失败清理，我们不需要复杂的依赖分析
				// 直接并行清理所有失败的进程即可
				cleanupSuccess := 0
				cleanupSkipped := 0
				cleanupFailed := 0
				
				// 使用并发清理失败的进程
				for _, proc := range failedProcesses {
					// 检查进程是否还在运行
					isRunning, err := process.IsRunning(proc)
					if err != nil || !isRunning {
						cleanupSkipped++
						continue
					}
					
					// 停止进程
					if err := process.Stop(proc); err != nil {
						fmt.Printf("⚠️ 清理进程 %s 失败: %v\n", proc.Name, err)
						cleanupFailed++
					} else {
						cleanupSuccess++
					}
				}
				
				if cleanupFailed > 0 {
					fmt.Printf("🧹 清理完成：成功 %d 个，失败 %d 个，跳过 %d 个（未运行）\n", 
						cleanupSuccess, cleanupFailed, cleanupSkipped)
				} else {
					fmt.Printf("🧹 清理完成：成功清理 %d 个进程，跳过 %d 个进程（未运行）\n", 
						cleanupSuccess, cleanupSkipped)
				}
			}

			// 如果有成功启动的进程，询问用户是否要回滚
			if len(successfulProcesses) > 0 {
				fmt.Printf("\n💡 提示：有 %d 个进程启动成功，如需全部回滚，请运行：\n", len(successfulProcesses))
				fmt.Print("   ./procmate stop ")
				for i, proc := range successfulProcesses {
					if i > 0 {
						fmt.Print(" ")
					}
					fmt.Print(proc.Name)
				}
				fmt.Println()
			}

			return fmt.Errorf("启动过程中有 %d 个进程失败", totalFailure)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}