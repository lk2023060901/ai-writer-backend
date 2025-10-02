package conf

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	MinIO     MinIOConfig
	Milvus    MilvusConfig
	Log       LogConfig
	MinerU    MinerUConfig
	Auth      AuthConfig
}

type ServerConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	GRPCPort int    `mapstructure:"grpc_port"`
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
}

type MilvusConfig struct {
	Host string
	Port int
}

type LogConfig struct {
	Level            string     `mapstructure:"level"`
	Format           string     `mapstructure:"format"`
	Output           string     `mapstructure:"output"`
	File             FileLogConfig `mapstructure:"file"`
	EnableCaller     bool       `mapstructure:"enablecaller"`
	EnableStacktrace bool       `mapstructure:"enablestacktrace"`
}

type FileLogConfig struct {
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"maxsize"`
	MaxAge     int    `mapstructure:"maxage"`
	MaxBackups int    `mapstructure:"maxbackups"`
	Compress   bool   `mapstructure:"compress"`
}

type MinerUConfig struct {
	BaseURL         string        `mapstructure:"base_url"`
	APIKey          string        `mapstructure:"api_key"`
	Timeout         time.Duration `mapstructure:"timeout"`
	MaxRetries      int           `mapstructure:"max_retries"`
	DefaultLanguage string        `mapstructure:"default_language"`
	EnableFormula   bool          `mapstructure:"enable_formula"`
	EnableTable     bool          `mapstructure:"enable_table"`
	ModelVersion    string        `mapstructure:"model_version"`
}

type AuthConfig struct {
	JWTSecret   string `mapstructure:"jwt_secret"`
	JWTIssuer   string `mapstructure:"jwt_issuer"`
	TOTPIssuer  string `mapstructure:"totp_issuer"`
	BackupCodes int    `mapstructure:"backup_codes"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}
