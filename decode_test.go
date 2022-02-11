package config

import (
	"path/filepath"
	"testing"
	"time"
)

type Driver string

type Config struct {
	Log map[string]string

	Net struct {
		Listen string

		TLS struct {
			Cert string
			Key  string
		}
	}

	Drivers []Driver

	Cache struct {
		Redis struct {
			Addr string
		}

		CleanupInterval time.Duration `config:"cleanup_interval"`
	}

	Store map[string]struct {
		Type  string
		Path  string
		Limit int64
	}
}

func Test_Decode(t *testing.T) {
	var cfg Config

	if err := Decode(&cfg, filepath.Join("testdata", "server.conf"), errh(t)); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg)
}
