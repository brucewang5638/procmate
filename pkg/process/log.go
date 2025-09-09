package process

import (
	"fmt"
	"os"
	"procmate/pkg/config"
	"sync"

	"github.com/hpcloud/tail"
)

// TailLog 查找、追踪并美化打印进程的日志
func TailLog(proc config.Process) error {
	var logFiles []string
	
	// 获取procmate管理的日志文件路径
	logFilePath, err := GetLogFile(proc)
	if err != nil {
		return fmt.Errorf("无法获取 '%s' 的日志文件路径: %w", proc.Name, err)
	}
	logFiles = append(logFiles, logFilePath)
	
	// 添加进程配置中指定的额外日志文件
	logFiles = append(logFiles, proc.LogFiles...)
	
	if len(logFiles) == 0 {
		fmt.Printf("📃 进程 '%s' 没有配置任何日志文件\n", proc.Name)
		return nil
	}

	// 检查并启动所有日志文件的追踪
	var wg sync.WaitGroup
	var tails []*tail.Tail
	
	for _, logFile := range logFiles {
		// 检查日志文件是否存在（仅作提示，不存在也会追踪等待创建）
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			fmt.Printf("📃 日志文件不存在，等待创建: %s\n", logFile)
		} else {
			fmt.Printf("📃 找到日志文件: %s\n", logFile)
		}

		// 使用 tail 库追踪日志文件
		t, err := tail.TailFile(logFile, tail.Config{
			ReOpen:    true,  // 文件被移动或删除时重新打开
			Follow:    true,  // 类似 tail -f
			MustExist: false, // 文件不存在时等待创建
		})

		if err != nil {
			fmt.Printf("⚠️ 无法开始追踪日志文件 '%s': %v\n", logFile, err)
			continue
		}
		
		tails = append(tails, t)
		
		wg.Add(1)
		go func(t *tail.Tail, filename string) {
			defer wg.Done()
			for line := range t.Lines {
				// 带文件名前缀打印日志行，方便区分来源
				fmt.Printf("[%s] %s\n", filename, line.Text)
			}
		}(t, logFile)
	}

	if len(tails) == 0 {
		return fmt.Errorf("无法追踪任何日志文件")
	}

	fmt.Printf("👀 正在追踪 '%s' 的 %d 个日志文件，按 Ctrl+C 退出\n", proc.Name, len(tails))

	// 等待所有goroutine完成
	wg.Wait()

	return nil
}
