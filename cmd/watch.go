package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"procmate/pkg/config"
	"procmate/pkg/process"

	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "启动守护模式，持续监控并自动重启已关闭的进程 🛡️",
	Long: `这是一个长期运行的命令。它会周期性地检查所有已启用进程的状态，
如果发现某个进程离线，则会自动尝试重新启动它。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("✅ procmate 守护模式已启动... (按 Ctrl+C 退出)")

		watchInterval := config.Cfg.Settings.WatchIntervalSec

		fmt.Printf("每 %d 秒检查一次所有已启用进程的状态。\n", watchInterval)

		// 创建定时器，每 watchInterval 秒触发一次
		ticker := time.NewTicker(time.Duration(watchInterval) * time.Second)

		defer ticker.Stop()

		// 捕获退出信号
		quitChannel := make(chan os.Signal, 1)
		signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)

		// 立即执行一次检查
		checkAndRestartProcesses()

		// 主循环
		for {
			select {
			case <-ticker.C:
				fmt.Println("\n⏰ [TICK] 周期性检查开始...")
				checkAndRestartProcesses()
			case <-quitChannel:
				fmt.Println("\n🛑 收到退出信号，正在关闭守护进程...")
				return nil
			}
		}
	},
}

// checkAndRestartProcesses 封装单次检查和重启逻辑
func checkAndRestartProcesses() {
	var needRestartProcesses []config.Process
	var timeoutProcesses []config.Process

	// 第一轮：检查所有进程状态，收集需要处理的进程
	for _, proc := range config.Cfg.Processes {
		if !proc.Enabled {
			continue
		}

		// 检查是否运行正常
		isRunning, _ := process.IsRunning(proc)
		isReady, err := process.IsReady(proc)
		if err != nil {
			fmt.Printf("❌ 检查进程就绪状态失败: %v\033[0m\n", err)
			continue
		}

		if isRunning {
			if isReady {
				// 绿色表示运行且就绪
				fmt.Printf("\033[32m✔️ 进程 '%s' 运行中且就绪\033[0m\n", proc.Name)
			} else {
				// 黄色表示运行但未就绪
				fmt.Printf("\033[33m♻️ 进程 '%s' 运行中，但未就绪\033[0m\n", proc.Name)

				// 检查是否超时，如果超时收集起来统一处理
				if isProcessTimeoutNoReady(proc) {
					timeoutProcesses = append(timeoutProcesses, proc)
				}
			}
		} else {
			// 红色 🚨 表示离线警告
			fmt.Printf("\033[31m🚨 警告: 进程 '%s' 离线！\033[0m\n", proc.Name)
			needRestartProcesses = append(needRestartProcesses, proc)
		}
	}

	// 第二轮：处理超时进程 - 停止它们
	if len(timeoutProcesses) > 0 {
		fmt.Printf("\n🚨 发现 %d 个超时进程，正在终止...\n", len(timeoutProcesses))
		for _, proc := range timeoutProcesses {
			if err := process.Stop(proc); err != nil {
				fmt.Printf("\033[31m❌ 终止超时进程 '%s' 失败: %v\033[0m\n", proc.Name, err)
			} else {
				fmt.Printf("\033[33m⚡ 超时进程 '%s' 已成功终止。\033[0m\n", proc.Name)
				// 将终止的超时进程也加入重启列表
				needRestartProcesses = append(needRestartProcesses, proc)
			}
		}
	}

	// 第三轮：并行重启需要重启的进程
	if len(needRestartProcesses) > 0 {
		fmt.Printf("\n⚡ 发现 %d 个离线进程，正在并行重启...\n", len(needRestartProcesses))
		
		// 构建进程映射表
		allProcessesMap := make(map[string]config.Process)
		for _, p := range config.Cfg.Processes {
			allProcessesMap[p.Name] = p
		}
		
		// 提取进程名称列表
		var processNames []string
		for _, proc := range needRestartProcesses {
			processNames = append(processNames, proc.Name)
		}
		
		// 获取分层执行计划
		executionLayers, err := process.GetExecutionLayers(allProcessesMap, processNames)
		if err != nil {
			fmt.Printf("\033[31m❌ 无法确定重启计划: %v\033[0m\n", err)
			return
		}

		if len(executionLayers) > 0 {
			fmt.Printf("📋 重启计划：共 %d 层，将并行重启:\n", len(executionLayers))
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

			// 使用智能失败处理的并行启动管理器执行重启
			manager := process.NewParallelStartManager(process.GetSmartParallelStartOptions())
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			
			layerResults, err := manager.StartProcessesInLayers(executionLayers, ctx)
			if err != nil {
				fmt.Printf("\033[31m❌ 并行重启失败: %v\033[0m\n", err)
				return
			}

			// 显示重启结果汇总
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
						fmt.Printf("\033[31m❌ 进程 %s 重启失败: %v\033[0m\n", result.Process.Name, result.Error)
					}
				}
			}
			
			fmt.Printf("\033[32m📊 重启结果：成功 %d 个，失败 %d 个，跳过 %d 个\033[0m\n", totalSuccess, totalFailure, totalSkipped)
		}
	}
}

// isProcessTimeoutNoReady 检查进程是否因为未就绪而超时
func isProcessTimeoutNoReady(proc config.Process) bool {
	// 计算超时时间
	timeoutSec := proc.StartTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = config.Cfg.Settings.DefaultStartTimeoutSec
	}
	if timeoutSec <= 0 {
		timeoutSec = 60 // 提供一个最终的默认值，防止两者都未配置
	}
	timeoutDuration := time.Duration(timeoutSec) * time.Second

	info, err := process.GetProcessInfo(proc)
	if err != nil {
		fmt.Printf("❌ 获取进程 '%s' 信息时发生意外错误: %v\n", proc.Name, err)
		return false
	}

	// 如果运行时间超过了配置的超时阈值，则认为进程卡住
	if info.Uptime > timeoutDuration {
		fmt.Printf("\033[31m🚨 进程 '%s' 运行已超过 %d 秒但仍未就绪，标记为超时\033[0m\n", proc.Name, int(timeoutDuration.Seconds()))
		return true
	}
	
	return false
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
