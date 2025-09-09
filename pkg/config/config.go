package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper" // 引入 viper 库
)

// Config 是整个配置文件的顶层结构。
type Config struct {
	Settings  Settings  `mapstructure:"settings"`
	Processes []Process `mapstructure:"processes"`
	Include   string    `mapstructure:"include"`
}

// Settings 结构体对应配置文件中的 'settings' 部分。
type Settings struct {
	RuntimeDir             string     `mapstructure:"runtime_dir"`
	DefaultStartTimeoutSec int        `mapstructure:"default_start_timeout_sec"`
	DefaultStopTimeoutSec  int        `mapstructure:"default_stop_timeout_sec"`
	WatchIntervalSec       int        `mapstructure:"watch_interval_sec"`
	LogOptions             LogOptions `mapstructure:"log_options"`
}

// Process 结构体对应 'processes' 列表中的每一个进程项。
type Process struct {
	Name    string `mapstructure:"name"`
	Group   string `mapstructure:"group"`
	Command string `mapstructure:"command"`
	WorkDir string `mapstructure:"workdir"`
	Port    int    `mapstructure:"port"`
	Enabled bool   `mapstructure:"enabled"`

	// 使用 int 表示超时（秒）
	// 如果 YAML 中未配置，将使用全局默认值。
	StartTimeoutSec int `mapstructure:"start_timeout_sec"`
	StopTimeoutSec  int `mapstructure:"stop_timeout_sec"`

	// 环境变量 (map 的键是环境变量名，值是其对应的值)
	Environment map[string]string `mapstructure:"environment"`

	// 依赖关系 (字符串切片)
	DependsOn []string `mapstructure:"depends_on"`

	// 额外的日志文件路径 (用于Java应用等使用日志框架的情况)
	LogFiles []string `mapstructure:"log_files"`
}

// LogOptions 结构体对应 'log_options' 部分，用于配置日志轮转。
type LogOptions struct {
	MaxSizeMB  int  `mapstructure:"max_size_mb"`
	MaxBackups int  `mapstructure:"max_backups"`
	MaxAgeDays int  `mapstructure:"max_age_days"`
	Compress   bool `mapstructure:"compress"`
	LocalTime  bool `mapstructure:"localTime"`
}

// Cfg 是一个指向 Config 实例的全局指针，用于在程序各处访问配置。
var Cfg *Config

// LoadConfig 使用 Viper 读取和解析配置文件。
// 最新版本支持 'include' 指令，并能处理重名进程（后来者覆盖）并发出警告。
func LoadConfig(path string) error {
	v := viper.New()
	v.SetConfigFile(path)

	// 读取主配置文件
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read main config: %w", err)
	}
	// 主配置反序列化入全局变量
	if err := v.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("failed to unmarshal main config: %w", err)
	}

	finalProcesses := make([]Process, 0)
	seenProcs := make(map[string]int) // K: 进程名, V: 在 finalProcesses 中的索引

	// 定义一个闭包，用于处理一个进程列表（来自主文件或 include 文件）
	process := func(procs []Process, sourceFile string) {
		for _, p := range procs {
			if index, exists := seenProcs[p.Name]; exists {
				fmt.Printf("Warning: Duplicate process '%s' in %s overwrites previous definition.\n", p.Name, sourceFile)
				finalProcesses[index] = p // 覆盖
			} else {
				finalProcesses = append(finalProcesses, p) // 追加
				seenProcs[p.Name] = len(finalProcesses) - 1
			}
		}
	}

	//  处理主配置文件中的进程
	process(Cfg.Processes, path)

	//  处理 include 文件
	if Cfg.Include != "" {
		globPath := filepath.Join(filepath.Dir(path), Cfg.Include)
		if files, err := filepath.Glob(globPath); err == nil {
			// 循环子配置文件
			for _, file := range files {
				includeViper := viper.New()
				includeViper.SetConfigFile(file)
				if includeViper.ReadInConfig() == nil {
					var tempCfg struct {
						Processes []Process `mapstructure:"processes"`
					}
					if includeViper.Unmarshal(&tempCfg) == nil {
						process(tempCfg.Processes, file)
					}
				}
			}
		}
	}

	// 3. 将最终结果赋回全局配置
	Cfg.Processes = finalProcesses
	return nil
}
