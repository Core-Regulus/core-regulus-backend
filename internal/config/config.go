package config

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
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

type CalendarConfig struct {
	Id string
	Service *calendar.Service	
}


type Config struct {
	Environment string
	SSH SSHConfig
	Database DatabaseConfig
	Calendar CalendarConfig
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

func loadCalendarConfig() {
	cfg.Calendar.Id = mustEnv("CALENDAR_ID")
		
	var data []byte
	var err error
	if (cfg.IsLocal()) {
		serviceAccountFile := "./.credentials/googleCalendar.json"
		data, err = os.ReadFile(serviceAccountFile)
		if err != nil {
			log.Fatalf("account file read error: %v", err)
		}
	}
	ctx := context.Background()
	
	var creds *google.Credentials
	creds, err = google.CredentialsFromJSON(ctx, data, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Can't load calendar account credentials: %v", err)
	}

	cfg.Calendar.Service, err = calendar.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Can't initialize Calendar API client: %v", err)
	}
}


func loadConfig() {
	cfg.Environment = getEnvironment();
	if (cfg.IsLocal()) {
		godotenv.Load(".env")	
		loadSSHConfig()		
	}
	loadDatabaseConfig()		
	loadCalendarConfig()
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
