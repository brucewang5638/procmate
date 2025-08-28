package cmd

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed /scripts/entity-utils.sh
var entityUtilsScript []byte

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start [all|component_name|service_name]...",
	Short: "启动一个或多个组件/服务",
	Long: `根据配置文件启动一个或多个实体。
可以指定 'all' 来启动所有已配置的实体，或提供一个或多个名称来启动特定的实体。`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("错误: 请至少提供一个要启动的实体名称，或使用 'all' 启动所有实体。")
			os.Exit(1)
		}

		var config Config
		if err := viper.Unmarshal(&config); err != nil {
			fmt.Printf("无法解码配置文件: %v\n", err)
			os.Exit(1)
		}

		targets := getTargetsToStart(args, &config)
		if len(targets) == 0 {
			fmt.Println("未找到需要启动的目标。请检查名称是否正确。")
			return
		}

		// 将嵌入的脚本写入临时文件
		scriptPath, err := writeTempScript(entityUtilsScript)
		if err != nil {
			fmt.Printf("无法创建临时脚本文件: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(scriptPath) // 确保执行完毕后删除临时脚本

		fmt.Printf("🚀 准备启动 %d 个实体...\n", len(targets))
		for _, target := range targets {
			startEntity(scriptPath, target)
		}
	},
}

// startEntity 调用 shell 脚本来启动单个实体
func startEntity(scriptPath string, entity interface{}) {
	var name, cmdStr, logFile, port string

	switch v := entity.(type) {
	case Component:
		name = v.Name
		cmdStr = v.StartCmd
		port = fmt.Sprintf("%d", v.Port)
		// logFile 可以根据需要从配置中读取或生成
		logFile = fmt.Sprintf("/var/log/hk/%s/component/%s.log", viper.GetString("projectName"), name)
	case Service:
		name = v.Name
		// 对于服务，我们需要动态构建启动命令
		cmdStr = fmt.Sprintf("java %s -jar %s", v.JvmOpts, v.JarPath)
		port = fmt.Sprintf("%d", v.Port)
		logFile = fmt.Sprintf("/var/log/hk/%s/service/%s.log", viper.GetString("projectName"), name)
	default:
		fmt.Printf("未知的实体类型\n")
		return
	}

	fmt.Printf("\n--- 正在处理: %s ---\n", name)
	// 构建命令: . /tmp/script.sh && start_entity "Name" "Command" "LogFile" "Port"
	fullCmd := fmt.Sprintf(". %s && start_entity '%s' \"%s\" '%s' '%s'", scriptPath, name, cmdStr, logFile, port)

	cmd := exec.Command("bash", "-c", fullCmd)
	// 将子进程的输出实时连接到当前进程的输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("执行启动命令失败 for %s: %v\n", name, err)
	}
}

// getTargetsToStart 根据输入参数解析出需要启动的实体列表
func getTargetsToStart(args []string, config *Config) []interface{} {
	var targets []interface{}
	allEntities := map[string]interface{}{}
	for _, c := range config.Components {
		allEntities[c.Name] = c
	}
	for _, s := range config.Services {
		allEntities[s.Name] = s
	}

	if len(args) == 1 && args[0] == "all" {
		for _, entity := range allEntities {
			targets = append(targets, entity)
		}
		return targets
	}

	for _, arg := range args {
		if entity, ok := allEntities[arg]; ok {
			targets = append(targets, entity)
		} else {
			fmt.Printf("警告: 在配置文件中未找到名为 '%s' 的实体，将跳过。\n", arg)
		}
	}
	return targets
}

// writeTempScript 将脚本内容写入一个可执行的临时文件
func writeTempScript(scriptContent []byte) (string, error) {
	tempFile, err := os.CreateTemp("", "hk-script-*.sh")
	if err != nil {
		return "", err
	}

	if _, err := tempFile.Write(scriptContent); err != nil {
		tempFile.Close()
		return "", err
	}
	tempFile.Close()

	// 使文件可执行
	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func init() {
	rootCmd.AddCommand(startCmd)
}
