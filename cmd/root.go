package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"procmate/pkg/config" // 引入我们自己写的 config 包

	"github.com/spf13/cobra" // 引入 cobra
)

// cfgFile 是一个包级私有变量，用于存储 --config 标志传入的配置文件路径。
var cfgFile string

// rootCmd 代表了我们应用的根命令。
// 当不带任何子命令直接调用应用时，执行的就是它。
var rootCmd = &cobra.Command{
	// Use 是命令的名称，也就是我们在终端里输入的内容
	Use: "procmate",
	// Short 是命令的简短描述，会出现在 'help' 列表里
	Short: "一个用于管理和监控进程的命令行工具",
	// Long 是命令的详细描述，当运行 'procmate help' 时会显示
	Long: `procmate 是一个用 Go编写的进程伴侣工具。
      它可以帮助您轻松地启动、停止、监控和管理在配置文件中定义的各种进程。`,
	// Run 字段是一个函数，如果这个命令有自己的执行逻辑（而不是必须跟一个子命令），
	// 那么这个函数就会被执行。对于根命令，我们通常让它打印帮助信息。
	Run: func(cmd *cobra.Command, args []string) {
		// 如果用户只输入了 'procmate' 而没有跟任何子命令，就打印帮助信息。
		cmd.Help()
	},
}

// Execute 函数是 rootCmd 的公共入口点。
// main.go 将会调用这个函数来启动整个命令处理流程。
func Execute() {
	// cobra 会解析命令行参数，找到对应的命令并执行。
	// 如果出错（比如，用户输入了不存在的标志），则会 panic 并打印错误。
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// init 函数在 main 函数之前自动执行。
// 我们在这里进行所有的初始化工作，比如定义标志和加载配置。
func init() {
	// cobra.OnInitialize() 注册一个或多个函数，这些函数会在解析标志之后
	// 执行命令的 Run 函数之前被 cobra 调用。我们在这里加载配置是最佳时机。
	cobra.OnInitialize(initConfig)

	// 这里我们为根命令定义了一个“持久化的标志”(Persistent Flag)。
	// 持久化意味着，这个标志不仅根命令可以用，所有子命令也都可以用。
	// &cfgFile: 将标志的值绑定到 cfgFile 变量上。
	// "config": 标志的全名 (--config)。
	// "f": 标志的短名 (-f)。
	// "": 默认值。
	// "配置文件路径 (默认为 ./config.yaml)": 帮助信息。
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "./config.yaml)")
}

// initConfig 函数是我们注册给 cobra.OnInitialize 的函数。
// 它负责调用我们之前写好的 config.LoadConfig 来加载配置。
func initConfig() {
	// 1. 如果用户通过 --config 标志提供了路径，则优先使用它。
	if cfgFile != "" {
		if err := config.LoadConfig(cfgFile); err != nil {
			fmt.Printf("加载指定的配置文件 %s 失败: %v\n", cfgFile, err)
			os.Exit(1)
		}
		// fmt.Printf("成功加载指定的配置文件: %s\n", cfgFile)
		return
	}

	// 2. 如果未指定路径，则按顺序搜索默认位置。
	//    - 当前工作目录
	//    - /etc/procmate/
	searchPaths := []string{
		"config.yaml",               // 当前工作目录
		"/etc/procmate/config.yaml", // 系统级配置目录
	}

	// (可选) 添加用户家目录的搜索路径
	if home, err := os.UserHomeDir(); err == nil {
		searchPaths = append(searchPaths, filepath.Join(home, ".config", "procmate", "config.yaml"))
	}

	// 3. 遍历搜索路径，找到第一个存在的配置文件并加载。
	for _, path := range searchPaths {
		// 使用 os.Stat 检查文件是否存在。
		if _, err := os.Stat(path); err == nil {
			if loadErr := config.LoadConfig(path); loadErr != nil {
				fmt.Printf("加载配置文件 %s 失败: %v\n", path, loadErr)
				os.Exit(1)
			}
			// (可选) 打印成功加载信息
			// fmt.Printf("成功加载配置文件: %s\n", path)
			return // 找到并成功加载后，立即返回
		}
	}

	// 4. 如果遍历完所有路径都找不到配置文件，则报错退出。
	fmt.Println("错误: 找不到配置文件。")
	fmt.Println("请在以下任一位置创建 config.yaml 文件，或使用 --config 标志指定路径:")
	for _, path := range searchPaths {
		fmt.Printf("  - %s\n", path)
	}
	os.Exit(1)
}
