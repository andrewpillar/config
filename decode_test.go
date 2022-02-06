package config

import (
	"path/filepath"
	"testing"
	"time"
)

type Driver string

type Config struct {
	Log map[string]string `config:"log"`

	Net struct {
		Listen string `config:"listen"`

		TLS struct {
			Cert string `config:"cert"`
			Key  string `config:"key"`
		} `config:"tls"`
	} `config:"net"`

	Drivers []Driver `config:"drivers"`

	Cache struct {
		Redis struct {
			Addr string `config:"addr"`
		} `config:"redis"`

		CleanupInterval time.Duration `config:"cleanup_interval"`
	} `config:"cache"`

	Store map[string]struct {
		Type  string `config:"type"`
		Path  string `config:"path"`
		Limit int64  `config:"limit"`
	} `config:"store"`
}

func Test_Decode(t *testing.T) {
	var cfg Config

	if err := Decode(&cfg, filepath.Join("testdata", "server.conf"), errh(t)); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg)
}
