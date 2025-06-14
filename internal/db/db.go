package db

import (
	"database/sql"
	"fmt"
	"io"
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

func ConnectViaSSH(cfg SSHPostgresConfig) (*sql.DB, error) {
	signer, err := ssh.ParsePrivateKey([]byte(cfg.PrivateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.SSHUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	sshAddr := fmt.Sprintf("%s:%d", cfg.SSHHost, cfg.SSHPort)
	sshClient, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH dial error: %w", err)
	}

	remoteAddr := fmt.Sprintf("%s:%d", cfg.DBHost, cfg.DBPort)
	localListener, err := net.Listen("tcp", "127.0.0.1:5433")
	if err != nil {
		return nil, fmt.Errorf("local port listen error: %w", err)
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

	dsn := fmt.Sprintf("host=127.0.0.1 port=5433 user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL open error: %w", err)
	}
	return db, nil
}