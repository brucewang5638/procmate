package cmd

import (
	"fmt"
	"os"
	"procmate/pkg/config"
	"procmate/pkg/process"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/spf13/cobra"
)

// statusCmd 代表 'procmate status' 命令
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "检查并显示所有已定义进程的状态 🔛",
	Long: `遍历配置文件中定义的所有进程，通过检查其PID文件和系统信息
来确定它们的详细运行时状态，并以表格形式显示结果。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 步骤 1: 遍历进程，将所有行数据收集到一个切片中
		var tableData [][]string
		for _, proc := range config.Cfg.Processes {
			if !proc.Enabled {
				continue
			}

			info, err := process.GetProcessInfo(proc)
			if err != nil {
				fmt.Fprintf(os.Stderr, "获取进程 '%s' 信息时发生意外错误: %v\n", proc.Name, err)
				continue
			}

			var row []string
			if info.IsRunning {
				var status = "♻️ RUNNING"

				if info.IsReady {
					status = "✅ READY"
				}

				portsStr := strings.Join(info.ListeningPorts, ",")
				if portsStr == "" {
					portsStr = "-"
				}
				row = []string{
					info.Name,
					fmt.Sprintf("%d", info.PID),
					status,
					info.Uptime.String(),
					fmt.Sprintf("%.1f%%", info.CPUPercent),
					fmt.Sprintf("%.1fMB", info.MemoryRSS),
					portsStr,
				}
			} else {
				status := "❌ OFFLINE"
				row = []string{
					info.Name,
					"-",
					status,
					"-",
					"-",
					"-",
					"-",
				}
			}
			tableData = append(tableData, row)
		}

		// 步骤 2: 完全按照示例的简洁风格进行渲染
		table := tablewriter.NewTable(os.Stdout,
			tablewriter.WithRenderer(renderer.NewMarkdown()),
		)
		table.Header("NAME", "PID", "STATUS", "UPTIME", "CPU%", "MEM(RSS)", "LISTENING")

		table.Bulk(tableData)

		table.Render()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
