package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	OldDB   DBConfig  `yaml:"old_db"`
	NewAPI  APIConfig `yaml:"new_api"`
	Options Options   `yaml:"options"`
}

// DBConfig 数据库配置
type DBConfig struct {
	Driver   string `yaml:"driver"`   // mysql, postgres, sqlite
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	Charset  string `yaml:"charset"`
	SSLMode  string `yaml:"ssl_mode"` // for postgres
}

// APIConfig API 配置
type APIConfig struct {
	BaseURL  string `yaml:"base_url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Options 迁移选项
type Options struct {
	RetryTimes    int    `yaml:"retry_times"`
	RetryDelay    int    `yaml:"retry_delay"`
	SkipExisting  bool   `yaml:"skip_existing"`
	MigrateCards  bool   `yaml:"migrate_cards"`
	OnlyActive    bool   `yaml:"only_active"`
	BatchSize     int    `yaml:"batch_size"`
	OldSitePath   string `yaml:"old_site_path"`
}

// CLIArgs 命令行参数
type CLIArgs struct {
	OldHost     string
	OldPort     int
	OldUser     string
	OldPassword string
	OldDatabase string
	OldDriver   string
	NewAPI      string
	NewUser     string
	NewPassword string
	NoSkip      bool
	NoCards     bool
	OldSitePath string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		OldDB: DBConfig{
			Driver:  "mysql",
			Host:    "127.0.0.1",
			Port:    3306,
			User:    "root",
			Password: "",
			Database: "dujiaoka",
			Charset:  "utf8mb4",
			SSLMode:  "disable",
		},
		NewAPI: APIConfig{
			BaseURL:  "http://127.0.0.1:8080/api/v1/admin",
			Username: "admin",
			Password: "admin123",
		},
		Options: Options{
			RetryTimes:   3,
			RetryDelay:   1,
			SkipExisting: true,
			MigrateCards: true,
			OnlyActive:   true,
			BatchSize:    500,
			OldSitePath:  "",
		},
	}
}

// LoadConfig 加载配置
func LoadConfig(configFile string, args *CLIArgs) (*Config, error) {
	cfg := DefaultConfig()

	// 从文件加载
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %w", err)
		}
	}

	// 命令行参数覆盖
	if args.OldHost != "" {
		cfg.OldDB.Host = args.OldHost
	}
	if args.OldPort > 0 {
		cfg.OldDB.Port = args.OldPort
	}
	if args.OldUser != "" {
		cfg.OldDB.User = args.OldUser
	}
	if args.OldPassword != "" {
		cfg.OldDB.Password = args.OldPassword
	}
	if args.OldDatabase != "" {
		cfg.OldDB.Database = args.OldDatabase
	}
	if args.OldDriver != "" {
		cfg.OldDB.Driver = args.OldDriver
	}
	if args.NewAPI != "" {
		cfg.NewAPI.BaseURL = args.NewAPI
	}
	if args.NewUser != "" {
		cfg.NewAPI.Username = args.NewUser
	}
	if args.NewPassword != "" {
		cfg.NewAPI.Password = args.NewPassword
	}
	if args.NoSkip {
		cfg.Options.SkipExisting = false
	}
	if args.NoCards {
		cfg.Options.MigrateCards = false
	}
	if args.OldSitePath != "" {
		cfg.Options.OldSitePath = args.OldSitePath
	}

	return cfg, nil
}

// GenerateSampleConfig 生成示例配置文件
func GenerateSampleConfig() {
	sample := `# 独角数卡迁移工具配置文件

# 老版数据库配置
old_db:
  driver: "mysql"          # 数据库驱动: mysql, postgres, sqlite
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "your_password"
  database: "dujiaoka"
  charset: "utf8mb4"
  ssl_mode: "disable"      # PostgreSQL SSL 模式: disable, require, verify-ca, verify-full

# SQLite 示例:
# old_db:
#   driver: "sqlite"
#   database: "/path/to/dujiaoka.db"

# PostgreSQL 示例:
# old_db:
#   driver: "postgres"
#   host: "127.0.0.1"
#   port: 5432
#   user: "postgres"
#   password: "your_password"
#   database: "dujiaoka"
#   ssl_mode: "disable"

# 新版 API 配置
new_api:
  base_url: "http://127.0.0.1:8080/api/v1/admin"
  username: "admin"
  password: "admin123"

# 迁移选项
options:
  retry_times: 3        # API 请求重试次数
  retry_delay: 1        # 重试间隔（秒）
  skip_existing: true   # 跳过已存在的数据
  migrate_cards: true   # 是否迁移卡密
  only_active: true     # 只迁移已启用的数据
  batch_size: 500       # 卡密批量导入大小
  old_site_path: ""     # 老版站点路径（用于图片迁移，如 /www/wwwroot/dujiaoka）
`
	fmt.Print(sample)
}
