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
	Email     EmailConfig
	OAuth2    OAuth2Config
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

type EmailConfig struct {
	SMTPHost       string        `mapstructure:"smtp_host"`
	SMTPPort       int           `mapstructure:"smtp_port"`
	FromAddr       string        `mapstructure:"from_addr"`
	FromName       string        `mapstructure:"from_name"`
	OAuth2Enabled  bool          `mapstructure:"oauth2_enabled"`
	MaxRetries     int           `mapstructure:"max_retries"`
	RetryInterval  time.Duration `mapstructure:"retry_interval"`
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
	SendTimeout    time.Duration `mapstructure:"send_timeout"`
}

type OAuth2Config struct {
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"`
	AuthURL      string   `mapstructure:"auth_url"`
	TokenURL     string   `mapstructure:"token_url"`
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
