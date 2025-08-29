package process

import (
	"fmt"
	"os"
	"procmate/pkg/config"

	"github.com/hpcloud/tail"
)

// TailLog 查找、追踪并美化打印进程的日志
func TailLog(proc config.Process) error {
	// 获取日志文件路径
	logFilePath, err := GetLogFile(proc)
	if err != nil {
		return fmt.Errorf("无法获取 '%s' 的日志文件路径: %w", proc.Name, err)
	}

	// 检查日志文件是否存在
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		fmt.Printf("📃 进程 '%s' 没有日志 (预期文件: %s)\n", proc.Name, logFilePath)
	}

	// 使用 tail 库追踪日志文件
	t, err := tail.TailFile(logFilePath, tail.Config{
		ReOpen:    true,  // 文件被移动或删除时重新打开
		Follow:    true,  // 类似 tail -f
		MustExist: false, // 文件不存在时等待创建
	})

	if err != nil {
		return fmt.Errorf("无法开始追踪日志文件 '%s': %w", logFilePath, err)
	}

	fmt.Printf("👀 正在追踪 '%s' 的日志，按 Ctrl+C 退出\n", proc.Name)

	// --- 步骤 3: 循环打印日志内容 ---
	// 从 tail 的通道中读取新的日志行
	for line := range t.Lines {
		// 直接打印从文件中读取到的原始文本行
		fmt.Println(line.Text)
	}

	return nil
}
