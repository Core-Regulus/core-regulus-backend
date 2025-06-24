package db

import (	
	"context"
	"core-regulus-backend/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/ssh"
)

type ConfigMap map[string]interface{}

var once sync.Once
var poolOnce sync.Once
var configOnce sync.Once
var mainPool *pgxpool.Pool
var dbConfig ConfigMap

func checkSSH() {
	once.Do(func() {
		cfg := config.Get()	
		if (cfg.IsLocal()) {
			connectSSH(cfg)	
		}		
	})	
}



func connectSSH(cfg *config.Config) {
	signer, err := ssh.ParsePrivateKey([]byte(cfg.SSH.PrivateKey))
	if err != nil {
		log.Fatalf("parse private key: %v", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.SSH.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	sshAddr := fmt.Sprintf("%s:%d", cfg.SSH.Host, cfg.SSH.Port)
	sshClient, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		log.Fatalf("SSH dial error: %v", err)
	}

	remoteAddr := fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port)
	localListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port))
	if err != nil {
		log.Fatalf("local port listen error: %v", err)
	}

	go func() {
		for {
			localConn, err := localListener.Accept()
			if err != nil {
				continue
			}
			go func() {
				defer localConn.Close()
				remoteConn, err := sshClient.Dial("tcp", remoteAddr)
				if err != nil {
					return
				}
				defer remoteConn.Close()

				go io.Copy(remoteConn, localConn)
				io.Copy(localConn, remoteConn)
			}()
		}
	}()
}

func connectDB(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
					cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name)

	log.Println(dsn)
	dbpool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}
	return dbpool, err
}

func Connect() *pgxpool.Pool {
	poolOnce.Do(func() {
		cfg := config.Get()			
		var err error
		checkSSH()
		mainPool, err = connectDB(cfg)		
		if err != nil {
			log.Fatal("Error:", err)
		}		
	})
	return mainPool
}

func Config() *ConfigMap {
	configOnce.Do(func() {
		pool := Connect()
		ctx := context.Background()
		var jsonData []byte
		err := pool.QueryRow(ctx, "select config.get()").Scan(&jsonData)
		if err != nil {
			log.Fatal("Query config error", err)
		}		
		if err := json.Unmarshal(jsonData, &dbConfig); err != nil {
			log.Fatal(err)
		}				
	})
	fmt.Printf("google calendar: %v\n", dbConfig["google.calendar"])
	return &dbConfig
}

func (c ConfigMap) GetString(key string) (string, bool) {
	if val, ok := c[key]; ok {
		switch v := val.(type) {
		case string:
			return v, true
		default:
			jsonBytes, err := json.Marshal(v)
			if err == nil {
				return string(jsonBytes), true
			}
		}
	}
	return "", false
}