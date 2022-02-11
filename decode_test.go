package config

import (
	"path/filepath"
	"testing"
	"time"
)

func Test_DecodeSimpleConfig(t *testing.T) {
	var cfg struct {
		Log map[string]string

		Net struct {
			Listen string

			TLS struct {
				Cert string
				Key  string
			}
		}

		Drivers []string

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

	if err := Decode(&cfg, filepath.Join("testdata", "server.conf"), errh(t)); err != nil {
		t.Fatal(err)
	}
}

func Test_DecodeArrays(t *testing.T) {
	var cfg struct {
		Strings   []string
		Ints      []int64
		Floats    []float64
		Bools     []bool
		Durations []time.Duration
		Sizes     []int64
	}

	if err := Decode(&cfg, filepath.Join("testdata", "array.conf"), errh(t)); err != nil {
		t.Fatal(err)
	}

	Strings := []string{"one", "two", "three", "four"}
	Ints := []int64{1, 2, 3, 4}
	Floats := []float64{1.2, 3.4, 5.6, 7.8}
	Bools := []bool{true, false}
	Durations := []time.Duration{time.Second, time.Minute * 2, time.Hour * 3, time.Hour * 4 * 24}
	Sizes := []int64{1, 2048, 3145728, 4294967296, 5497558138880}

	for i, str := range Strings {
		if cfg.Strings[i] != str {
			t.Errorf("Strings[%d] - unexpected string, expected=%q, got=%q\n", i, str, cfg.Strings[i])
		}
	}

	for i, i64 := range Ints {
		if cfg.Ints[i] != i64 {
			t.Errorf("Ints[%d] - unexpected int64, expected=%d, got=%d\n", i, i64, cfg.Ints[i])
		}
	}

	for i, f64 := range Floats {
		if cfg.Floats[i] != f64 {
			t.Errorf("Floats[%d] - unexpected float64, expected=%f, got=%f\n", i, f64, cfg.Floats[i])
		}
	}

	for i, b := range Bools {
		if cfg.Bools[i] != b {
			t.Errorf("Bools[%d] - unexpected bool, expected=%v, got=%v\n", i, b, cfg.Bools[i])
		}
	}

	for i, dur := range Durations {
		if cfg.Durations[i] != dur {
			t.Errorf("Durations[%d] - unexpected time.Duration, expected=%v, got=%v\n", i, dur, cfg.Durations[i])
		}
	}

	for i, siz := range Sizes {
		if cfg.Sizes[i] != siz {
			t.Errorf("Sizes[%d] - unexpected int64, expected=%d, got=%d\n", i, siz, cfg.Sizes[i])
		}
	}
}

func Test_DecodeLabel(t *testing.T) {
	type TLS struct {
		CA string
	}

	type Auth struct {
		Addr string

		TLS TLS
	}

	var cfg struct {
		Auth map[string]Auth
	}

	if err := Decode(&cfg, filepath.Join("testdata", "label.conf"), errh(t)); err != nil {
		t.Fatal(err)
	}

	expected := map[string]Auth{
		"internal": {
			Addr: "postgres://localhost:5432",
			TLS:  TLS{},
		},
		"ldap": {
			Addr: "ldap://example.com",
			TLS:  TLS{CA: "/var/lib/ssl/ca.crt"},
		},
		"saml": {
			Addr: "https://idp.example.com",
			TLS:  TLS{CA: "/var/lib/ssl/ca.crt"},
		},
	}

	for label, auth := range expected {
		cfg, ok := cfg.Auth[label]

		if !ok {
			t.Fatalf("could not find label %q\n", label)
		}

		if cfg.Addr != auth.Addr {
			t.Fatalf("unexpected Addr, expected=%q, got=%q\n", cfg.Addr, auth.Addr)
		}

		if cfg.TLS.CA != auth.TLS.CA {
			t.Fatalf("unexpected TLS.CA, expected=%q, got=%q\n", cfg.TLS.CA, auth.TLS.CA)
		}
	}
}
