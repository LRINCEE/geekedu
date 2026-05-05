package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Server      ServerConfig
	Database    DatabaseConfig
	OSS         OSSConfig
	Redis       RedisConfig
	JWT         JWTConfig
	LogicServer LogicServerConfig
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type ServerConfig struct {
	HttpPort string `mapstructure:"http_port"`
	GrpcPort string `mapstructure:"grpc_port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type OSSConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	BucketName      string `mapstructure:"bucket_name"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type LogicServerConfig struct {
	Addr string `mapstructure:"addr"`
}

var Cfg *AppConfig

func InitConfig(path string) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	// Bind environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Env var bindings
	viper.BindEnv("server.http_port", "HTTP_PORT")
	viper.BindEnv("server.grpc_port", "GRPC_PORT")
	viper.BindEnv("database.host", "DB_HOST")
	viper.BindEnv("database.port", "DB_PORT")
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")
	viper.BindEnv("database.dbname", "DB_NAME")
	viper.BindEnv("redis.host", "REDIS_HOST")
	viper.BindEnv("redis.port", "REDIS_PORT")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.db", "REDIS_DB")
	viper.BindEnv("oss.endpoint", "OSS_ENDPOINT")
	viper.BindEnv("oss.access_key_id", "OSS_ACCESS_KEY")
	viper.BindEnv("oss.access_key_secret", "OSS_SECRET_KEY")
	viper.BindEnv("oss.bucket_name", "OSS_BUCKET")
	viper.BindEnv("jwt.secret", "JWT_SECRET")
	viper.BindEnv("logicserver.addr", "LOGIC_SERVER_ADDR")

	// Defaults
	viper.SetDefault("server.http_port", "8080")
	viper.SetDefault("server.grpc_port", "9090")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "3306")
	viper.SetDefault("database.user", "root")
	viper.SetDefault("database.password", "root")
	viper.SetDefault("database.dbname", "geekedu")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("jwt.secret", "geekedu-jwt-secret")
	viper.SetDefault("jwt.expire_hours", 24)
	viper.SetDefault("logicserver.addr", "localhost:9090")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Config file not found or error reading: %v, using env/defaults", err)
	}

	Cfg = &AppConfig{}
	if err := viper.Unmarshal(Cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}
}

func GetConfig() *AppConfig {
	if Cfg == nil {
		log.Fatal("Config not initialized. Call InitConfig first.")
	}
	return Cfg
}
