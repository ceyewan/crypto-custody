package config

import (
	"github.com/spf13/viper"
)

// Config 应用程序配置结构
type Config struct {
	Debug       bool   `mapstructure:"debug"`
	Port        string `mapstructure:"port"`
	TempDir     string `mapstructure:"temp_dir"`
	BinDir      string `mapstructure:"bin_dir"`
	KeygenBin   string `mapstructure:"keygen_bin"`
	SigningBin  string `mapstructure:"signing_bin"`
	ManagerAddr string `mapstructure:"manager_addr"`
	ManagerPort string `mapstructure:"manager_port"`
}

// LoadConfig 从配置文件加载配置
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 设置默认值
	viper.SetDefault("debug", false)
	viper.SetDefault("port", "8080")
	viper.SetDefault("temp_dir", "./temp")
	viper.SetDefault("bin_dir", "./bin")
	viper.SetDefault("keygen_bin", "gg20_keygen")
	viper.SetDefault("signing_bin", "gg20_signing")
	viper.SetDefault("manager_addr", "http://127.0.0.1:8081")

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
