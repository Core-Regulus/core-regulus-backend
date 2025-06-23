package db

import (
	"context"
	"core-regulus-backend/internal/config"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/ssh"
)
var once sync.Once
var poolOnce sync.Once
var mainPool *pgxpool.Pool

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