package process

import (
	"context"
	"fmt"
	"sync"
	"time"

	"procmate/pkg/config"
)

// StopResult 表示进程停止的结果
type StopResult struct {
	Process  config.Process // 进程配置
	Success  bool           // 停止是否成功
	Error    error          // 停止失败时的错误信息
	Duration time.Duration  // 停止耗时
	WasRunning bool         // 进程是否原本在运行
}

// StopLayerResult 表示一层进程停止的结果
type StopLayerResult struct {
	LayerIndex   int           // 层级索引（从顶层开始）
	Results      []StopResult  // 该层所有进程的停止结果
	SuccessCount int           // 停止成功的进程数量
	FailureCount int           // 停止失败的进程数量
	SkippedCount int           // 跳过的进程数量（未运行）
	Duration     time.Duration // 整层停止耗时
	HasFailures  bool          // 是否存在停止失败的进程
}

// ParallelStopManager 并行停止管理器
type ParallelStopManager struct {
	maxConcurrency int           // 最大并发数，0表示无限制
	layerTimeout   time.Duration // 单层停止超时时间
	processTimeout time.Duration // 单个进程停止超时时间
	showProgress   bool          // 是否显示进度信息
}

// ParallelStopOptions 并行停止配置选项
type ParallelStopOptions struct {
	MaxConcurrency int           // 最大并发数（0 = 无限制）
	LayerTimeout   time.Duration // 单层停止超时时间（0 = 使用进程配置）
	ProcessTimeout time.Duration // 单个进程停止超时时间（0 = 使用进程配置）
	ShowProgress   bool          // 是否显示停止进度
}

// NewParallelStopManager 创建新的并行停止管理器
func NewParallelStopManager(options ParallelStopOptions) *ParallelStopManager {
	// 设置默认值
	if options.LayerTimeout == 0 {
		options.LayerTimeout = 5 * time.Minute // 默认5分钟层超时
	}
	if options.ProcessTimeout == 0 {
		options.ProcessTimeout = 1 * time.Minute // 默认1分钟进程超时
	}

	return &ParallelStopManager{
		maxConcurrency: options.MaxConcurrency,
		layerTimeout:   options.LayerTimeout,
		processTimeout: options.ProcessTimeout,
		showProgress:   options.ShowProgress,
	}
}

// StopProcessesInLayers 按层并行停止进程（反向停止，从上层开始）
func (m *ParallelStopManager) StopProcessesInLayers(layers [][]config.Process, ctx context.Context) ([]StopLayerResult, error) {
	var allResults []StopLayerResult

	if m.showProgress {
		fmt.Printf("🛑 开始分层并行停止，共 %d 层\n", len(layers))
	}

	// 逆序处理层级（从顶层开始停止）
	for i := len(layers) - 1; i >= 0; i-- {
		layerIndex := len(layers) - 1 - i // 计算显示用的层级索引
		layer := layers[i]

		if m.showProgress {
			fmt.Printf("\n📋 停止第 %d/%d 层，包含 %d 个进程...\n", layerIndex+1, len(layers), len(layer))
		}

		// 创建该层的上下文，设置超时
		layerCtx, cancel := context.WithTimeout(ctx, m.layerTimeout)
		
		// 停止当前层的所有进程
		layerResult := m.stopLayer(layerCtx, layerIndex, layer)
		cancel() // 确保释放上下文资源
		
		allResults = append(allResults, layerResult)

		// 检查上下文是否被取消
		if err := ctx.Err(); err != nil {
			return allResults, fmt.Errorf("停止操作被取消: %w", err)
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
		fmt.Printf("\n🎯 所有层停止完成！总计 成功: %d，失败: %d，跳过: %d\n", 
			totalSuccess, totalFailure, totalSkipped)
	}

	return allResults, nil
}

// stopLayer 停止单个层中的所有进程
func (m *ParallelStopManager) stopLayer(ctx context.Context, layerIndex int, processes []config.Process) StopLayerResult {
	startTime := time.Now()
	
	layerResult := StopLayerResult{
		LayerIndex: layerIndex,
		Results:    make([]StopResult, len(processes)),
	}

	// 创建工作通道和结果通道
	jobs := make(chan int, len(processes))
	results := make(chan struct {
		index int
		result StopResult
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
				
				// 停止单个进程
				result := m.stopSingleProcess(ctx, process)
				
				// 发送结果
				select {
				case results <- struct {
					index int
					result StopResult
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
		if !result.result.WasRunning {
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

// stopSingleProcess 停止单个进程
func (m *ParallelStopManager) stopSingleProcess(ctx context.Context, process config.Process) StopResult {
	startTime := time.Now()
	
	result := StopResult{
		Process: process,
	}

	// 检查进程是否在运行
	isRunning, err := IsRunning(process)
	if err != nil || !isRunning {
		result.Success = true
		result.WasRunning = false
		result.Duration = time.Since(startTime)
		return result
	}
	
	result.WasRunning = true

	// 创建进程停止的上下文，设置超时
	processTimeout := m.processTimeout
	if process.StopTimeoutSec > 0 {
		processTimeout = time.Duration(process.StopTimeoutSec) * time.Second
	}
	
	processCtx, cancel := context.WithTimeout(ctx, processTimeout)
	defer cancel()

	// 在协程中停止进程，以支持超时控制
	done := make(chan error, 1)
	go func() {
		done <- Stop(process)
	}()

	// 等待进程停止完成或超时
	select {
	case err := <-done:
		result.Duration = time.Since(startTime)
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("停止进程 '%s' 失败: %w", process.Name, err)
		} else {
			result.Success = true
		}
	case <-processCtx.Done():
		result.Duration = time.Since(startTime)
		result.Success = false
		result.Error = fmt.Errorf("停止进程 '%s' 超时 (%.1fs)", process.Name, result.Duration.Seconds())
	}

	return result
}

// GetDefaultParallelStopOptions 获取默认的并行停止配置
func GetDefaultParallelStopOptions() ParallelStopOptions {
	return ParallelStopOptions{
		MaxConcurrency: 0,                // 无并发限制
		LayerTimeout:   5 * time.Minute,  // 5分钟层超时
		ProcessTimeout: 1 * time.Minute,  // 1分钟进程超时
		ShowProgress:   true,             // 显示进度
	}
}

// GetAggressiveParallelStopOptions 获取激进的并行停止配置
func GetAggressiveParallelStopOptions() ParallelStopOptions {
	return ParallelStopOptions{
		MaxConcurrency: 0,                // 无并发限制
		LayerTimeout:   2 * time.Minute,  // 较短的层超时
		ProcessTimeout: 30 * time.Second, // 较短的进程超时
		ShowProgress:   true,             // 显示进度
	}
}