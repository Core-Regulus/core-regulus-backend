package config

import (	
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"github.com/joho/godotenv"	
)

type SSHConfig struct {
	PrivateKey string
	Host string
	Port int
	User string
}

type DatabaseConfig struct {
	Host string
	Port int
	Name string
	User string
	Password string	
}

type Config struct {
	Environment string
	SSH SSHConfig
	Database DatabaseConfig
}

func (c Config) IsLocal() bool {
	return c.Environment == "local"
}

var cfg Config
var once sync.Once

func getEnvironment() string {
	env := os.Getenv("environment")
	if (env != "") {
		return env
	}
	return "local"	
}

func loadSSHConfig() {
	cfg.SSH.PrivateKey = strings.ReplaceAll(mustEnv("SSH_PRIVATE_KEY"), `\n`, "\n")
	cfg.SSH.Host = mustEnv("SSH_HOST")
	cfg.SSH.Port,_ = strconv.Atoi(mustEnv("SSH_PORT"))
	cfg.SSH.User = mustEnv("SSH_USER")
}

func loadDatabaseConfig() {	
	cfg.Database.Port,_ = strconv.Atoi(mustEnv("DB_PORT"))
	cfg.Database.User = mustEnv("DB_USER")
	cfg.Database.Password = mustEnv("DB_PASSWORD")
	cfg.Database.Name = mustEnv("DB_NAME")
	cfg.Database.Host = mustEnv("DB_HOST")
}

func loadConfig() {
	cfg.Environment = getEnvironment();
	if (cfg.IsLocal()) {
		godotenv.Load(".env")	
		loadSSHConfig()		
	}
	loadDatabaseConfig()			
}

func Get() *Config {
	once.Do(func() {
		loadConfig()
	})
	return &cfg
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("missing env var: %s", key)
	}
	return val
}
