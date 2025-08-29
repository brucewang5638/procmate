// pkg/process/log_handler.go

package process

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"procmate/pkg/config"
)

// LogEntry 定义了结构化日志的格式
type LogEntry struct {
	Timestamp string `json:"timestamp"` // 日志时间，ISO 8601 格式
	App       string `json:"app"`       // 应用名称
	Stream    string `json:"stream"`    // 日志来源：stdout 或 stderr
	Message   string `json:"message"`   // 日志内容
}

// handleLogStream 读取输入流中的日志行，将其封装为结构化日志，并写入每日日志文件
func handleLogStream(proc config.Process, streamName string, stream io.Reader) {
	appName := proc.Name
	scanner := bufio.NewScanner(stream) // 创建 Scanner 来按行读取输入流
	for scanner.Scan() {
		line := scanner.Text() // 读取一行日志

		// 将日志封装为结构化格式
		entry := LogEntry{
			Timestamp: time.Now().UTC().Format(time.RFC3339), // 当前 UTC 时间
			App:       appName,
			Stream:    streamName,
			Message:   line,
		}

		// 将结构体转换为 JSON 格式
		jsonLine, err := json.Marshal(entry)
		if err != nil {
			// 序列化失败时，直接输出到 stderr
			fmt.Fprintf(os.Stderr, "FATAL: 无法序列化日志条目 %s: %v\n", appName, err)
			continue
		}

		// 获取当天日志文件路径
		logFile, err := GetLogFile(proc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取日志文件失败 %s: %v\n", appName, err)
			continue
		}

		// 以追加模式打开日志文件，如果不存在则创建
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "打开日志文件失败 %s: %v\n", logFile, err)
			continue
		}

		// 将 JSON 日志写入文件，并确保文件在写入后关闭
		func() {
			defer f.Close() // 确保文件关闭，防止资源泄漏
			if _, err := f.Write(append(jsonLine, '\n')); err != nil {
				fmt.Fprintf(os.Stderr, "写入日志文件失败 %s: %v\n", logFile, err)
			}
		}()
	}

	// 检查扫描过程中是否有错误发生
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "读取日志流错误 %s: %v\n", appName, err)
	}
}
