package config

import (
	"github.com/spf13/viper" // 引入 viper 库
)

// Process 定义了一个需要被管理的进程的结构。
// 字段上的 `mapstructure:"..."` 标签（tag）是给 Viper 使用的，
// 它告诉 Viper 在解析 YAML 文件时，应该把哪个键的值赋给这个字段。
// 例如，YAML 中的 `name:` 键会对应到 `Name` 字段。
type Process struct {
	Name    string `mapstructure:"name"`
	Group   string `mapstructure:"group"`
	Command string `mapstructure:"command"`
	WorkDir string `mapstructure:"workdir"`
	Port    int    `mapstructure:"port"`
	Enabled bool   `mapstructure:"enabled"`
}

// Config 是整个配置文件的顶层结构。
// 它包含一个 Process 类型的切片（slice），切片是 Go 中表示动态数组的方式。
type Config struct {
	Processes []Process `mapstructure:"processes"`
}

// 全局变量，用于在内存中缓存解析后的配置。
// 使用指针 *Config 是为了在多个地方共享同一份配置数据，避免不必要的拷贝。
var Cfg *Config

// LoadConfig 函数负责读取和解析配置文件。
func LoadConfig(path string) error {
	// 1. 初始化 Viper
	v := viper.New()

	// 2. 设置配置文件路径和名称
	v.SetConfigFile(path)   // 直接指定完整配置文件路径
	v.SetConfigType("yaml") // 明确指定配置文件类型为 YAML

	// 3. 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 如果读取文件出错，将错误返回
		return err
	}

	// 4. 将配置反序列化到 Cfg 全局变量中
	//    v.Unmarshal() 会解析配置并将其内容填充到传入的指针指向的结构体中。
	//    我们传入 &Cfg，因为 Cfg本身是一个指针，我们需要传递它的地址，
	//    以便 Unmarshal 能够修改 Cfg使其指向一个新的、填充好数据的 Config 实例。
	if err := v.Unmarshal(&Cfg); err != nil {
		// 如果解析配置出错，将错误返回
		return err
	}

	return nil // 一切顺利，返回 nil (表示没有错误)
}
