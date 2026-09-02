package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Mysql  Mysql
	Server Server
	Jwt    Jwt
	Redis  Redis
	Email  Email
	MiniO  MiniO
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

type Email struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Pass     string `mapstructure:"pass"`
	Expire   int    `mapstructure:"expire"`
	CoolDown int    `mapstructure:"cool_down"`
}

type MiniO struct {
	MinioEndpoint  string `mapstructure:"minio_endpoint"`
	MinioAccessKey string `mapstructure:"minio_access_key"`
	MinioSecretKey string `mapstructure:"minio_secret_key"`
	MinioUseSsl    bool   `mapstructure:"minio_use_ssl"`
	MinioBucket    string `mapstructure:"minio_bucket"`
	MinioPublicURL string `mapstructure:"minio_public_url"`
	MaxFileSize    int64  `mapstructure:"max_file_size"`
	MaxImageSize   int64  `mapstructure:"max_image_size"`
	ExpireTime     int64  `mapstructure:"expire_time"`
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
