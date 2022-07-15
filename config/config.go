package config

import (
	"fmt"
	"github.com/spf13/viper"
)

//InitConfig 初始化配置文件
func InitConfig() (err error) {
	viper.New()

	viper.SetConfigName("config")
	viper.AddConfigPath("./config/")
	viper.SetConfigType("json")

	if err = viper.ReadInConfig(); err != nil {
		return
	}

	fmt.Println("初始化配置文件成功")
	return nil
}
