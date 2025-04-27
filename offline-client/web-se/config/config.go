package config

import (
	"github.com/spf13/viper"
)

// Config 应用程序配置结构
type Config struct {
	Debug          bool   `mapstructure:"debug"`            // SE 是否启用调试模式
	CardReaderName string `mapstructure:"card_reader_name"` // SE 名称
	Port           string `mapstructure:"port"`
	TempDir        string `mapstructure:"temp_dir"`
	BinDir         string `mapstructure:"bin_dir"`
	KeygenBin      string `mapstructure:"keygen_bin"`
	SigningBin     string `mapstructure:"signing_bin"`
	ManagerAddr    string `mapstructure:"manager_addr"`
	ManagerPort    string `mapstructure:"manager_port"`
	// 日志配置
	LogDir        string `mapstructure:"log_dir"`         // 日志目录
	LogFile       string `mapstructure:"log_file"`        // 日志文件名
	LogMaxSize    int    `mapstructure:"log_max_size"`    // 单个日志文件最大大小(MB)
	LogMaxBackups int    `mapstructure:"log_max_backups"` // 保留的旧日志文件数量
	LogMaxAge     int    `mapstructure:"log_max_age"`     // 日志文件保留天数
	LogCompress   bool   `mapstructure:"log_compress"`    // 是否压缩旧日志
}

// LoadConfig 从配置文件加载配置
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 设置默认值
	viper.SetDefault("debug", false)
	viper.SetDefault("card_reader_name", "")
	viper.SetDefault("port", "8080")
	viper.SetDefault("temp_dir", "./temp")
	viper.SetDefault("bin_dir", "./bin")
	viper.SetDefault("keygen_bin", "gg20_keygen")
	viper.SetDefault("signing_bin", "gg20_signing")
	viper.SetDefault("manager_addr", "http://127.0.0.1:8000")
	// 日志默认值
	viper.SetDefault("log_dir", "./logs")
	viper.SetDefault("log_file", "web-se.log")
	viper.SetDefault("log_max_size", 10)    // 10MB
	viper.SetDefault("log_max_backups", 10) // 保留10个旧文件
	viper.SetDefault("log_max_age", 30)     // 保留30天
	viper.SetDefault("log_compress", true)  // 压缩旧日志文件

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在时使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
