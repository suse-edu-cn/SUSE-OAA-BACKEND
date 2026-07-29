package config

import "github.com/spf13/viper"

type Config struct {
	Mysql  Mysql
	Server Server
	Jwt    Jwt
	Redis  Redis
}

type Mysql struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}
type Server struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type Jwt struct {
	Secret      string `mapstructure:"secret"`
	ExpireHour  int    `mapstructure:"expire_hour"`
	RefreshTime uint   `mapstructure:"refresh_time"`
}
type Redis struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
}

func ConfigInit() Config {
	var config Config
	viper.SetConfigName("config")
	viper.AddConfigPath("../configs")
	viper.AddConfigPath("./configs")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	if err := viper.Unmarshal(&config); err != nil {
		panic(err)
	}
	return config
}
