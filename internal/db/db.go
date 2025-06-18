package db

import (
	"core-regulus-backend/internal/config"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"time"	
	_ "github.com/lib/pq"
	"golang.org/x/crypto/ssh"
)

type SSHPostgresConfig struct {
	SSHUser       string
	SSHHost       string
	SSHPort       int
	PrivateKeyPEM string

	DBUser     string
	DBPassword string
	DBName     string
	DBHost     string
	DBPort     int
}

func ConnectSSH(cfg *config.Config) {
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

	time.Sleep(300 * time.Millisecond)	
}

func ConnectDB(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
					cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL open error: %w", err)
	}
	return db, nil
}

func Connect() {
	cfg := config.Get()	
	var conn *sql.DB
	var err error
	if (cfg.IsLocal()) {
		ConnectSSH(cfg)	
	}
	conn, err = ConnectDB(cfg)		
	if err != nil {
		log.Fatal("Error:", err)
	}
	defer conn.Close()

	var version string
	err = conn.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("PostgreSQL version:", version)
}