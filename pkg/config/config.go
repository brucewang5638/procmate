package config

import (
	"github.com/spf13/viper" // 引入 viper 库
)

// Settings 结构体对应配置文件中的 'settings' 部分。
type Settings struct {
	RuntimeDir             string     `mapstructure:"runtime_dir"`
	DefaultStartTimeoutSec int        `mapstructure:"default_start_timeout_sec"`
	DefaultStopTimeoutSec  int        `mapstructure:"default_stop_timeout_sec"`
	WatchIntervalSec       int        `mapstructure:"watch_interval_sec"`
	LogOptions             LogOptions `mapstructure:"log_options"`
}

// LogOptions 结构体对应 'log_options' 部分，用于配置日志轮转。
type LogOptions struct {
	MaxSizeMB  int  `mapstructure:"max_size_mb"`
	MaxBackups int  `mapstructure:"max_backups"`
	MaxAgeDays int  `mapstructure:"max_age_days"`
	Compress   bool `mapstructure:"compress"`
	LocalTime  bool `mapstructure:"localTime"`
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
}

// Config 是整个配置文件的顶层结构。
type Config struct {
	Settings  Settings  `mapstructure:"settings"`
	Processes []Process `mapstructure:"processes"`
}

// Cfg 是一个指向 Config 实例的全局指针，用于在程序各处访问配置。
var Cfg *Config

// LoadConfig 使用 Viper 读取和解析配置文件。
// 注意：Viper 的 Unmarshal 会自动处理新字段。
func LoadConfig(path string) error {
	v := viper.New()

	// 2. 设置配置文件路径和名称
	v.SetConfigFile(path)   // 直接指定完整配置文件路径
	v.SetConfigType("yaml") // 明确指定配置文件类型为 YAML

	// 3. 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return err
	}

	// 4. 将配置反序列化到 Cfg 全局变量中
	//    v.Unmarshal() 会解析配置并将其内容填充到传入的指针指向的结构体中。
	//    我们传入 &Cfg，因为 Cfg本身是一个指针，我们需要传递它的地址，
	//    以便 Unmarshal 能够修改 Cfg使其指向一个新的、填充好数据的 Config 实例。
	if err := v.Unmarshal(&Cfg); err != nil {
		return err
	}

	return nil
}
