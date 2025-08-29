package process

import (
	"encoding/json"
	"fmt"
	"os"
	"procmate/pkg/config"
	"time"

	"github.com/fatih/color"
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
		fmt.Printf("⏳ 今日 '%s' 没有日志 (预期文件: %s)\n", proc.Name, logFilePath)
		fmt.Println("等待新日志生成...")
	}

	// --- 创建不同场景的颜色打印机 ---
	colorTime := color.New(color.FgWhite).Add(color.Faint) // 时间戳用灰色
	colorApp := color.New(color.FgCyan)                    // 应用名用青色
	colorStdout := color.New(color.FgGreen)                // stdout 用绿色
	colorStderr := color.New(color.FgRed)                  // stderr 用红色

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

	// --- 循环处理每一行日志 ---
	for line := range t.Lines {
		var entry LogEntry
		// 尝试将行文本解析为 JSON
		if err := json.Unmarshal([]byte(line.Text), &entry); err == nil {
			// --- 解析成功，进行美化输出 ---

			// 1. 打印时间戳和应用名
			parsedTime, err := time.Parse(time.RFC3339, entry.Timestamp)
			if err != nil {
				colorTime.Printf("[%s] ", entry.Timestamp) // 解析失败则打印原始时间
			} else {
				colorTime.Printf("[%s] ", parsedTime.Format("15:04:05")) // 只显示时分秒
			}
			colorApp.Printf("[%s] ", entry.App)

			// 2. 根据日志流 (stdout/stderr) 选择不同颜色打印消息
			if entry.Stream == "stderr" {
				colorStderr.Printf("[stderr]: %s\n", entry.Message)
			} else {
				colorStdout.Printf("[stdout]: %s\n", entry.Message)
			}

		} else {
			// --- 解析失败，直接打印原文 ---
			// 保证对非 JSON 格式日志的兼容性
			fmt.Println(line.Text)
		}
	}

	return nil
}
