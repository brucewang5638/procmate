package process

import (
	"context"
	"fmt"
	"sync"
	"time"

	"procmate/pkg/config"
)

// StartupResult 表示进程启动的结果
// 包含启动状态、错误信息和相关进程信息
type StartupResult struct {
	Process   config.Process // 进程配置
	Success   bool          // 启动是否成功
	Error     error         // 启动失败时的错误信息
	Duration  time.Duration // 启动耗时
	PID       int          // 进程ID（启动成功时）
	IsSkipped bool         // 是否因为已运行而跳过启动
}

// LayerResult 表示一层进程启动的结果
// 包含该层所有进程的启动结果和整体统计信息
type LayerResult struct {
	LayerIndex    int             // 层级索引（从0开始）
	Results       []StartupResult // 该层所有进程的启动结果
	SuccessCount  int             // 启动成功的进程数量
	FailureCount  int             // 启动失败的进程数量
	SkippedCount  int             // 因已运行而跳过的进程数量
	Duration      time.Duration   // 整层启动耗时
	HasFailures   bool            // 是否存在启动失败的进程
}

// ParallelStartManager 并行启动管理器
// 负责管理进程的并行启动，包括依赖关系处理、错误管理、超时控制等
type ParallelStartManager struct {
	maxConcurrency    int           // 最大并发数，0表示无限制
	layerTimeout      time.Duration // 单层启动超时时间
	processTimeout    time.Duration // 单个进程启动超时时间
	stopOnFirstError  bool          // 是否在首个错误时停止启动
	enableRollback    bool          // 是否启用失败回滚
	showProgress      bool          // 是否显示进度信息
}

// ParallelStartOptions 并行启动配置选项
type ParallelStartOptions struct {
	MaxConcurrency   int           // 最大并发数（0 = 无限制）
	LayerTimeout     time.Duration // 单层启动超时时间（0 = 使用进程配置）
	ProcessTimeout   time.Duration // 单个进程启动超时时间（0 = 使用进程配置）
	StopOnFirstError bool          // 遇到第一个错误时是否停止
	EnableRollback   bool          // 是否在失败时回滚已启动的进程
	ShowProgress     bool          // 是否显示启动进度
}

// NewParallelStartManager 创建新的并行启动管理器
//
// 参数:
//   - options: 并行启动配置选项
//
// 返回:
//   - *ParallelStartManager: 配置好的并行启动管理器实例
//
// 示例:
//   manager := NewParallelStartManager(ParallelStartOptions{
//       MaxConcurrency: 5,
//       LayerTimeout: 5 * time.Minute,
//       StopOnFirstError: true,
//       EnableRollback: true,
//   })
func NewParallelStartManager(options ParallelStartOptions) *ParallelStartManager {
	// 设置默认值
	if options.LayerTimeout == 0 {
		options.LayerTimeout = 10 * time.Minute // 默认10分钟层超时
	}
	if options.ProcessTimeout == 0 {
		options.ProcessTimeout = 2 * time.Minute // 默认2分钟进程超时
	}

	return &ParallelStartManager{
		maxConcurrency:   options.MaxConcurrency,
		layerTimeout:     options.LayerTimeout,
		processTimeout:   options.ProcessTimeout,
		stopOnFirstError: options.StopOnFirstError,
		enableRollback:   options.EnableRollback,
		showProgress:     options.ShowProgress,
	}
}

// StartProcessesInLayers 按层并行启动进程
// 这是并行启动管理器的核心方法，按依赖关系分层启动进程
//
// 参数:
//   - layers: 分层执行计划，每层包含可并行执行的进程列表
//   - ctx: 上下文，用于取消操作
//
// 返回:
//   - []LayerResult: 每层的启动结果详情
//   - error: 启动过程中的致命错误
//
// 工作流程:
//   1. 逐层处理进程启动
//   2. 同层内进程并行启动
//   3. 等待每层完成后再进入下一层
//   4. 根据配置决定是否在失败时停止或回滚
//
// 示例:
//   ctx := context.Background()
//   results, err := manager.StartProcessesInLayers(layers, ctx)
//   if err != nil {
//       log.Fatal("并行启动失败:", err)
//   }
func (m *ParallelStartManager) StartProcessesInLayers(layers [][]config.Process, ctx context.Context) ([]LayerResult, error) {
	var allResults []LayerResult
	var startedProcesses []config.Process // 用于失败时的回滚

	if m.showProgress {
		fmt.Printf("🚀 开始分层并行启动，共 %d 层\n", len(layers))
	}

	// 逐层处理
	for layerIndex, layer := range layers {
		if m.showProgress {
			fmt.Printf("\n📋 启动第 %d/%d 层，包含 %d 个进程...\n", layerIndex+1, len(layers), len(layer))
		}

		// 创建该层的上下文，设置超时
		layerCtx, cancel := context.WithTimeout(ctx, m.layerTimeout)
		
		// 启动当前层的所有进程
		layerResult := m.startLayer(layerCtx, layerIndex, layer)
		cancel() // 确保释放上下文资源
		
		allResults = append(allResults, layerResult)

		// 记录成功启动的进程，用于可能的回滚
		for _, result := range layerResult.Results {
			if result.Success && !result.IsSkipped {
				startedProcesses = append(startedProcesses, result.Process)
			}
		}

		// 检查是否需要因为错误而停止
		if layerResult.HasFailures {
			if m.stopOnFirstError {
				if m.showProgress {
					fmt.Printf("❌ 第 %d 层存在启动失败，停止后续启动\n", layerIndex+1)
				}
				
				// 如果启用了回滚，回滚已启动的进程
				if m.enableRollback {
					if err := m.rollbackStartedProcesses(startedProcesses); err != nil {
						return allResults, fmt.Errorf("启动失败且回滚失败: %w", err)
					}
					if m.showProgress {
						fmt.Println("✅ 已回滚所有已启动的进程")
					}
				}
				
				return allResults, fmt.Errorf("第 %d 层存在 %d 个进程启动失败", layerIndex+1, layerResult.FailureCount)
			} else {
				if m.showProgress {
					fmt.Printf("⚠️  第 %d 层存在 %d 个失败，但继续启动后续层\n", layerIndex+1, layerResult.FailureCount)
				}
			}
		}

		// 检查上下文是否被取消
		if err := ctx.Err(); err != nil {
			return allResults, fmt.Errorf("启动被取消: %w", err)
		}

		if m.showProgress {
			fmt.Printf("✅ 第 %d 层完成，成功: %d，失败: %d，跳过: %d\n", 
				layerIndex+1, layerResult.SuccessCount, layerResult.FailureCount, layerResult.SkippedCount)
		}
	}

	if m.showProgress {
		totalSuccess := 0
		totalFailure := 0
		totalSkipped := 0
		for _, result := range allResults {
			totalSuccess += result.SuccessCount
			totalFailure += result.FailureCount
			totalSkipped += result.SkippedCount
		}
		fmt.Printf("\n🎯 所有层启动完成！总计 成功: %d，失败: %d，跳过: %d\n", 
			totalSuccess, totalFailure, totalSkipped)
	}

	return allResults, nil
}

// startLayer 启动单个层中的所有进程
// 在该层内并行执行所有进程的启动操作
func (m *ParallelStartManager) startLayer(ctx context.Context, layerIndex int, processes []config.Process) LayerResult {
	startTime := time.Now()
	
	layerResult := LayerResult{
		LayerIndex: layerIndex,
		Results:    make([]StartupResult, len(processes)),
	}

	// 创建工作通道和结果通道
	jobs := make(chan int, len(processes))
	results := make(chan struct {
		index int
		result StartupResult
	}, len(processes))

	// 确定并发工作协程数量
	workerCount := len(processes)
	if m.maxConcurrency > 0 && workerCount > m.maxConcurrency {
		workerCount = m.maxConcurrency
	}

	// 启动工作协程池
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for jobIndex := range jobs {
				process := processes[jobIndex]
				
				// 启动单个进程
				result := m.startSingleProcess(ctx, process)
				
				// 发送结果
				select {
				case results <- struct {
					index int
					result StartupResult
				}{jobIndex, result}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// 分发任务
	go func() {
		defer close(jobs)
		for i := range processes {
			select {
			case jobs <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	// 收集结果
	go func() {
		wg.Wait()
		close(results)
	}()

	// 处理结果
	for result := range results {
		layerResult.Results[result.index] = result.result
		
		// 更新统计信息
		if result.result.IsSkipped {
			layerResult.SkippedCount++
		} else if result.result.Success {
			layerResult.SuccessCount++
		} else {
			layerResult.FailureCount++
			layerResult.HasFailures = true
		}
	}

	layerResult.Duration = time.Since(startTime)
	
	return layerResult
}

// startSingleProcess 启动单个进程
// 处理单个进程的启动逻辑，包括运行检查、启动操作、错误处理
func (m *ParallelStartManager) startSingleProcess(ctx context.Context, process config.Process) StartupResult {
	startTime := time.Now()
	
	result := StartupResult{
		Process: process,
	}

	// 检查进程是否已在运行
	isRunning, err := IsRunning(process)
	if err == nil && isRunning {
		// 进一步检查是否已就绪
		if isReady, _ := IsReady(process); isReady {
			result.Success = true
			result.IsSkipped = true
			result.Duration = time.Since(startTime)
			return result
		}
	}

	// 创建进程启动的上下文，设置超时
	processTimeout := m.processTimeout
	if process.StartTimeoutSec > 0 {
		processTimeout = time.Duration(process.StartTimeoutSec) * time.Second
	}
	
	processCtx, cancel := context.WithTimeout(ctx, processTimeout)
	defer cancel()

	// 在协程中启动进程，以支持超时控制
	done := make(chan error, 1)
	go func() {
		done <- Start(process)
	}()

	// 等待进程启动完成或超时
	select {
	case err := <-done:
		result.Duration = time.Since(startTime)
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("启动进程 '%s' 失败: %w", process.Name, err)
		} else {
			result.Success = true
			// 获取进程PID（如果可能）
			if pid, err := ReadPid(process); err == nil {
				result.PID = pid
			}
		}
	case <-processCtx.Done():
		result.Duration = time.Since(startTime)
		result.Success = false
		result.Error = fmt.Errorf("启动进程 '%s' 超时 (%.1fs)", process.Name, result.Duration.Seconds())
	}

	return result
}

// rollbackStartedProcesses 回滚已启动的进程
// 在启动失败时，停止之前已经成功启动的进程
func (m *ParallelStartManager) rollbackStartedProcesses(processes []config.Process) error {
	if len(processes) == 0 {
		return nil
	}

	if m.showProgress {
		fmt.Printf("🔄 开始回滚 %d 个已启动的进程...\n", len(processes))
	}

	var rollbackErrors []error
	
	// 反向停止进程（后启动的先停止）
	for i := len(processes) - 1; i >= 0; i-- {
		process := processes[i]
		
		if m.showProgress {
			fmt.Printf("⏹️  停止进程 %s...\n", process.Name)
		}
		
		if err := Stop(process); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("停止进程 '%s' 失败: %w", process.Name, err))
		}
	}

	if len(rollbackErrors) > 0 {
		return fmt.Errorf("回滚过程中遇到 %d 个错误: %v", len(rollbackErrors), rollbackErrors)
	}

	return nil
}

// GetDefaultParallelStartOptions 获取默认的并行启动配置
// 提供合理的默认配置，适用于大多数场景
//
// 返回:
//   - ParallelStartOptions: 默认的并行启动配置
func GetDefaultParallelStartOptions() ParallelStartOptions {
	return ParallelStartOptions{
		MaxConcurrency:   0,                  // 无并发限制
		LayerTimeout:     10 * time.Minute,   // 10分钟层超时
		ProcessTimeout:   2 * time.Minute,    // 2分钟进程超时
		StopOnFirstError: true,               // 遇错停止
		EnableRollback:   true,               // 启用回滚
		ShowProgress:     true,               // 显示进度
	}
}

// GetConservativeParallelStartOptions 获取保守的并行启动配置
// 提供更保守的配置，适用于生产环境或不稳定的系统
//
// 返回:
//   - ParallelStartOptions: 保守的并行启动配置
func GetConservativeParallelStartOptions() ParallelStartOptions {
	return ParallelStartOptions{
		MaxConcurrency:   3,                  // 限制并发数
		LayerTimeout:     15 * time.Minute,   // 更长的层超时
		ProcessTimeout:   5 * time.Minute,    // 更长的进程超时
		StopOnFirstError: true,               // 遇错停止
		EnableRollback:   true,               // 启用回滚
		ShowProgress:     true,               // 显示进度
	}
}

// GetAggressiveParallelStartOptions 获取激进的并行启动配置
// 提供更激进的配置，适用于快速启动场景
//
// 返回:
//   - ParallelStartOptions: 激进的并行启动配置
func GetAggressiveParallelStartOptions() ParallelStartOptions {
	return ParallelStartOptions{
		MaxConcurrency:   0,                  // 无并发限制
		LayerTimeout:     5 * time.Minute,    // 较短的层超时
		ProcessTimeout:   30 * time.Second,   // 较短的进程超时
		StopOnFirstError: false,              // 不因错误停止
		EnableRollback:   false,              // 不启用回滚
		ShowProgress:     true,               // 显示进度
	}
}