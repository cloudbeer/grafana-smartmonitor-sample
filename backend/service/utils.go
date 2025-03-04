package service

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBName        string
	DBUser        string
	DBPassword    string
	Headless      bool
	AdminPassword string
	Front         string
	ChromeDP      string
	BRUrl         string
	BRAPIKey      string
}

var logger = log.New(os.Stdout, "service|", log.LstdFlags)
var globalConfig *Config

func GetGlobalConfig() *Config {
	return globalConfig
}

func GetEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

// LoadConfig 从多个来源加载配置信息，优先级为：命令行参数 > 环境变量 > .env文件 > 默认值
func LoadConfig() (*Config, error) {
	// 初始化配置结构
	cfg := &Config{
		// 设置默认值
		DBHost:        "localhost",
		DBPort:        "3306",
		DBName:        "gr-bd",
		DBUser:        "root",
		DBPassword:    "tester",
		Front:         "http://localhost:3000",
		ChromeDP:      "ws://localhost:9222",
		AdminPassword: "admin",
		Headless:      true,
		BRUrl:         "",
		BRAPIKey:      "",
	}

	// 1. 尝试加载 .env 文件（开发环境）
	if err := godotenv.Load(); err != nil {
		logger.Printf("未找到 .env 文件或无法读取: %v", err)
		logger.Printf("将从系统环境变量获取配置（生产环境模式）")
	} else {
		logger.Printf("成功从 .env 文件加载配置（开发环境模式）")
	}

	// 2. 从环境变量加载配置（覆盖默认值）
	envMappings := map[string]*string{
		"DB_HOST":                &cfg.DBHost,
		"DB_PORT":                &cfg.DBPort,
		"DB_NAME":                &cfg.DBName,
		"DB_USER":                &cfg.DBUser,
		"DB_PASSWORD":            &cfg.DBPassword,
		"FRONT_URL":              &cfg.Front,
		"CHROME_DP_URL":          &cfg.ChromeDP,
		"GRAFANA_ADMIN_PASSWORD": &cfg.AdminPassword,
		"BR_URL":                 &cfg.BRUrl,
		"BR_API_KEY":             &cfg.BRAPIKey,
	}

	for envKey, configPtr := range envMappings {
		if value := os.Getenv(envKey); value != "" {
			*configPtr = value
		}
	}

	// 处理布尔类型的环境变量
	if headless := os.Getenv("HEADLESS"); headless != "" {
		cfg.Headless, _ = strconv.ParseBool(headless)
	}

	// 4. 验证必要的配置是否已设置
	if cfg.DBHost == "" || cfg.DBPort == "" || cfg.DBName == "" || cfg.DBUser == "" {
		return nil, fmt.Errorf("必要的数据库配置值未设置")
	}

	// 5. 打印最终配置信息（不包含敏感信息）
	logger.Println("---- Configuration -------")
	logger.Printf("DBHost=%s, DBPort=%s, DBName=%s, DBUser=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser)
	logger.Printf("ChromeDP=%s, Headless=%t, Front=%s, BRUrl=%s",
		cfg.ChromeDP, cfg.Headless, cfg.Front, cfg.BRUrl)
	logger.Println("-------------------------")

	// 6. 设置全局配置并返回
	globalConfig = cfg
	return cfg, nil
}

func parseDateTime(str string) (time.Time, error) {
	// Adjust the layout string based on your MySQL date/time format
	layout := "2006-01-02 15:04:05"
	return time.Parse(layout, str)
}

func ReplaceHost(connectionURL, urlStr string) string {
	origin, err := url.Parse(connectionURL)

	if err != nil {
		return urlStr // return the original URL if it's invalid
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr // return the original URL if it's invalid
	}

	u.Scheme = origin.Scheme
	u.Host = origin.Host

	return u.String()
}
