package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

// rootCmd 代表了我们应用的基础命令，没有任何子命令时被调用
var rootCmd = &cobra.Command{
	Use:   "hk-console",
	Short: "一个统一管理和监控组件、服务的工具",
	Long: `hk-console 是一个现代化的命令行工具，旨在简化服务的启停、
状态监控和日志查看。它通过一个简单的配置文件来管理所有实体。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("欢迎使用 hk-console！")
		fmt.Println("请使用 'hk-console --help' 查看所有可用命令。")
	},
}

// Execute 将所有子命令添加到根命令中，并设置相应的标志。
// 这是 main.main() 调用的主要函数。
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认是 ./config.yaml)")
}

// initConfig 在 Cobra 初始化时被调用，用于读取配置文件
func initConfig() {
	if cfgFile != "" {
		// 使用命令行标志指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 否则，在当前目录查找名为 config.yaml 的文件
		vip.AddConfigPath(".")
		vip.SetConfigName("config")
		vip.SetConfigType("yaml")
	}

	vip.AutomaticEnv() // 读取匹配的环境变量

	// 如果能找到并读取配置文件，则使用它
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("成功加载配置文件:", viper.ConfigFileUsed())
	} else {
		fmt.Println("警告: 无法加载配置文件。", err)
	}
}
