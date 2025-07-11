package config

import (
	"crypto/rsa"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
)

type SSHConfig struct {
	PrivateKey string
	Host string
	Port int
	User string
}

type JWTConfig struct {
	PrivateKey *rsa.PrivateKey
	PublicKey *rsa.PublicKey
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
	JWT JWTConfig
}

func (c Config) IsLocal() bool {
	return c.Environment == "local"
}

var cfg Config
var once sync.Once

func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
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

func loadJWTConfig() {
	privateKey := strings.ReplaceAll(mustEnv("JWT_PRIVATE_KEY"), `\n`, "\n")
	publicKey := strings.ReplaceAll(mustEnv("JWT_PUBLIC_KEY"), `\n`, "\n")

	var privateKeyErr error
	cfg.JWT.PrivateKey, privateKeyErr = jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKey))
	if privateKeyErr != nil {
		log.Fatalf("Can't decode private key %v", privateKeyErr)
	}
	var publicKeyErr error
	cfg.JWT.PublicKey, publicKeyErr = jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
	if publicKeyErr != nil {
		log.Fatalf("Can't decode public key %v", publicKeyErr)
	}			
}

func loadConfig() {
	cfg.Environment = getEnvironment();
	if (cfg.IsLocal()) {
		godotenv.Load(".env")	
		loadSSHConfig()		
	}
	loadDatabaseConfig()			
	loadJWTConfig()
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
