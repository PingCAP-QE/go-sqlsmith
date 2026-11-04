// Package dbutil shares connection safety between the CLI tools.
package dbutil

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
)

type Options struct {
	Host, User, Password, PasswordFile, Database, Socket string
	Port                                                 int
	TLS                                                  bool
	TLSCA, TLSServerName                                 string
	AllowEmptyPassword                                   bool
	AllowInsecureTransport                               bool
}

// Configs returns bootstrap and database configurations. The driver applies sql_mode to
// every physical connection, including reconnects and worker connections.
func Configs(o Options) (*mysql.Config, *mysql.Config, error) {
	password := o.Password
	if o.PasswordFile != "" {
		if password != "" {
			return nil, nil, fmt.Errorf("use either -passwd or -password-file")
		}
		data, err := os.ReadFile(o.PasswordFile)
		if err != nil {
			return nil, nil, fmt.Errorf("read password file: %w", err)
		}
		password = strings.TrimSuffix(string(data), "\n")
		if strings.HasSuffix(string(data), "\r\n") {
			password = strings.TrimSuffix(password, "\r")
		}
	} else if password == "" {
		password = os.Getenv("SQLSMITH_PASSWORD")
	}
	if password == "" && !o.AllowEmptyPassword {
		return nil, nil, fmt.Errorf("empty database password: set SQLSMITH_PASSWORD, -password-file or -passwd; disposable passwordless instances require -allow-empty-password")
	}
	if o.Database == "" {
		return nil, nil, fmt.Errorf("database name must not be empty")
	}
	if !o.TLS && (o.TLSCA != "" || o.TLSServerName != "") {
		return nil, nil, fmt.Errorf("-tls-ca and -tls-server-name require -tls")
	}
	if !o.TLS && o.Socket == "" && !o.AllowInsecureTransport {
		ip := net.ParseIP(o.Host)
		if !strings.EqualFold(o.Host, "localhost") && (ip == nil || !ip.IsLoopback()) {
			return nil, nil, fmt.Errorf("remote TCP connections require -tls; isolated plaintext test networks require -allow-insecure-transport")
		}
	}

	cfg := mysql.NewConfig()
	cfg.User, cfg.Passwd = o.User, password
	cfg.Net, cfg.Addr = "tcp", net.JoinHostPort(o.Host, strconv.Itoa(o.Port))
	if o.Socket != "" {
		cfg.Net, cfg.Addr = "unix", o.Socket
	}
	// Preserve the generator's existing SQL semantics without a global SET.
	cfg.Params = map[string]string{"sql_mode": "'NO_ENGINE_SUBSTITUTION'"}
	if o.TLS {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: o.TLSServerName}
		if o.TLSCA != "" {
			pem, err := os.ReadFile(o.TLSCA)
			if err != nil {
				return nil, nil, fmt.Errorf("read TLS CA: %w", err)
			}
			roots, err := x509.SystemCertPool()
			if err != nil {
				roots = x509.NewCertPool()
			}
			if !roots.AppendCertsFromPEM(pem) {
				return nil, nil, fmt.Errorf("TLS CA file contains no certificates")
			}
			tlsConfig.RootCAs = roots
		}
		cfg.TLS = tlsConfig
	}
	bootstrap := cfg.Clone()
	cfg.DBName = o.Database
	return bootstrap, cfg, nil
}

// Open avoids a DSN string round trip: mysql v1.7 cannot escape database names
// containing query delimiters. NewConnector passes the database as data.
func Open(cfg *mysql.Config) (*sql.DB, error) {
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(connector), nil
}

// Target returns audit context without authentication material.
func Target(cfg *mysql.Config) string {
	return fmt.Sprintf("user=%q network=%q address=%q database=%q tls=%t", cfg.User, cfg.Net, cfg.Addr, cfg.DBName, cfg.TLS != nil)
}
