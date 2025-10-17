package process

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"procmate/pkg/config"
	"strings"
	"sync"

	"github.com/hpcloud/tail"
)

const (
	defaultLastLines = 100
	seekChunkSize    = 1024
)

// findLastNLinesStart calculates an offset at the beginning of the Nth last line of a file.
// It reads the file from the end in chunks to avoid loading the entire file into memory.
func findLastNLinesStart(filename string, n int) (int64, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return 0, err
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return 0, nil
	}

	var offset int64 = fileSize
	var lineCount = 0
	buf := make([]byte, seekChunkSize)

	for {
		seekPos := offset - seekChunkSize
		if seekPos < 0 {
			seekPos = 0
		}

		readSize, err := f.ReadAt(buf, seekPos)
		if err != nil && err != io.EOF {
			return 0, err
		}

		for i := readSize - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				lineCount++
				if lineCount >= n {
					return seekPos + int64(i) + 1, nil
				}
			}
		}

		offset = seekPos
		if offset == 0 {
			break
		}
	}

	return 0, nil
}

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

		// Determine the starting position for tailing.
		var location *tail.SeekInfo
		if stat, err := os.Stat(logFile); err == nil && stat.Size() > 0 {
			start, err := findLastNLinesStart(logFile, defaultLastLines)
			if err == nil {
				location = &tail.SeekInfo{Offset: start, Whence: io.SeekStart}
			} else {
				fmt.Printf("⚠️ 无法定位到最近 %d 行日志, 将从头开始: %v\n", defaultLastLines, err)
			}
		}

		// 使用 tail 库追踪日志文件
		t, err := tail.TailFile(logFile, tail.Config{
			ReOpen:    true,                  // 文件被移动或删除时重新打开
			Follow:    true,                  // 类似 tail -f
			MustExist: false,                 // 文件不存在时等待创建
			Location:  location,              // 从文件末尾或指定行数开始
			Logger:    tail.DiscardingLogger, // 禁止 tail 库自身的日志输出
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
				prefix := getLogPrefix(filename, len(logFiles) > 1)
				if prefix != "" {
					fmt.Printf("[%s] %s\n", prefix, line.Text)
				} else {
					fmt.Println(line.Text)
				}
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

// getLogPrefix 根据文件路径生成合适的日志前缀
func getLogPrefix(filename string, multipleFiles bool) string {
	// 如果只有一个文件，不显示前缀
	if !multipleFiles {
		return ""
	}

	// 检查是否是procmate管理的日志文件
	if strings.Contains(filename, "/logs/") && strings.HasSuffix(filename, ".log") {
		// 对于procmate日志，使用进程名而不是通用的"procmate"标识
		baseName := filepath.Base(filename)
		processName := strings.TrimSuffix(baseName, ".log")
		return "stdout/" + processName // 表示这是进程的stdout/stderr输出
	}

	// 对于用户自定义的日志文件，使用完整文件名
	baseName := filepath.Base(filename)
	return "file/" + baseName
}
