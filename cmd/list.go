package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// --- 配置数据结构定义 ---

// Config 是整个配置文件的根结构
type Config struct {
	ProjectName string      `mapstructure:"projectName"`
	LogDir      string      `mapstructure:"logDir"`
	Components  []Component `mapstructure:"components"`
	Services    []Service   `mapstructure:"services"`
}

// Component 定义了一个组件的属性
type Component struct {
	Name       string   `mapstructure:"name"`
	EntityType string   `mapstructure:"entityType"`
	Port       int      `mapstructure:"port"`
	StartCmd   string   `mapstructure:"startCmd"`
	Tags       []string `mapstructure:"tags"`
}

// Service 定义了一个服务 (Jar) 的属性
type Service struct {
	Name      string `mapstructure:"name"`
	EntityType string `mapstructure:"entityType"`
	Port      int    `mapstructure:"port"`
	JarPath   string `mapstructure:"jarPath"`
	JvmOpts   string `mapstructure:"jvmOpts"`
}

// listCmd 代表 'list' 子命令
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有已配置的组件和服务",
	Long:  `读取配置文件，并以表格形式清晰地展示所有已配置的实体及其关键信息。`,
	Run: func(cmd *cobra.Command, args []string) {
		var config Config
		// 将 viper 加载的配置数据解码到我们定义的 Config 结构体中
		if err := viper.Unmarshal(&config); err != nil {
			fmt.Printf("无法解码配置文件: %v\n", err)
			os.Exit(1)
		}

		// 使用 tabwriter 来创建对齐的表格输出
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

		if len(config.Components) > 0 {
			fmt.Fprintln(w, "--- 组件 (Components) ---")
			fmt.Fprintln(w, "名称\t端口\t启动命令")
			fmt.Fprintln(w, "----\t----\t--------")
			for _, c := range config.Components {
				fmt.Fprintf(w, "%s\t%d\t%s\n", c.Name, c.Port, c.StartCmd)
			}
			fmt.Fprintln(w, "") // 空行
		}

		if len(config.Services) > 0 {
			fmt.Fprintln(w, "--- 服务 (Services) ---")
			fmt.Fprintln(w, "名称\t端口\tJAR 路径")
			fmt.Fprintln(w, "----\t----\t--------")
			for _, s := range config.Services {
				fmt.Fprintf(w, "%s\t%d\t%s\n", s.Name, s.Port, s.JarPath)
			}
		}

		w.Flush()
	},
}

func init() {
	// 将 listCmd 添加到 rootCmd，这样它就成为了一个可用的子命令
	rootCmd.AddCommand(listCmd)
}
