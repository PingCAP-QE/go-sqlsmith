package dbutil

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPasswordSourcesAndSessionConfiguration(t *testing.T) {
	t.Setenv("SQLSMITH_PASSWORD", "env-secret")
	opts := Options{Host: "::1", Port: 4000, User: "limited", Database: "db?tls=skip-verify"}
	bootstrap, cfg, err := Configs(opts)
	require.NoError(t, err)
	require.Equal(t, "env-secret", cfg.Passwd)
	require.Equal(t, opts.Database, cfg.DBName)
	require.Equal(t, "[::1]:4000", cfg.Addr)
	require.Equal(t, "'NO_ENGINE_SUBSTITUTION'", cfg.Params["sql_mode"])
	require.False(t, cfg.MultiStatements)
	require.Nil(t, cfg.TLS)
	require.Empty(t, bootstrap.DBName)
	require.Equal(t, cfg.Params, bootstrap.Params)
	require.NotContains(t, Target(cfg), "env-secret")
	opts.Password = "explicit-secret"
	_, cfg, err = Configs(opts)
	require.NoError(t, err)
	require.Equal(t, "explicit-secret", cfg.Passwd)
	opts.Password = ""
	opts.PasswordFile = filepath.Join(t.TempDir(), "password")
	require.NoError(t, os.WriteFile(opts.PasswordFile, []byte("file:@/? secret\r\n"), 0600))
	_, cfg, err = Configs(opts)
	require.NoError(t, err)
	require.Equal(t, "file:@/? secret", cfg.Passwd)
	opts.PasswordFile = ""
	t.Setenv("SQLSMITH_PASSWORD", "")
	_, _, err = Configs(opts)
	require.Error(t, err)
	opts.AllowEmptyPassword = true
	_, _, err = Configs(opts)
	require.NoError(t, err)
}

func TestVerifiedTLSConfiguration(t *testing.T) {
	opts := Options{Host: "127.0.0.1", Port: 4000, User: "limited", Password: "test", Database: "test", TLS: true, TLSServerName: "database.example"}
	_, cfg, err := Configs(opts)
	require.NoError(t, err)
	require.NotNil(t, cfg.TLS)
	require.False(t, cfg.TLS.InsecureSkipVerify)
	require.Equal(t, uint16(tls.VersionTLS12), cfg.TLS.MinVersion)
	require.Equal(t, opts.TLSServerName, cfg.TLS.ServerName)
	require.False(t, cfg.AllowFallbackToPlaintext)
	opts.TLS = false
	_, _, err = Configs(opts)
	require.Error(t, err)
}

func TestRemotePlaintextNeedsOptIn(t *testing.T) {
	opts := Options{Host: "remote.example", Port: 4000, User: "limited", Password: "test", Database: "test"}
	_, _, err := Configs(opts)
	require.Error(t, err)
	opts.AllowInsecureTransport = true
	_, _, err = Configs(opts)
	require.NoError(t, err)
}
