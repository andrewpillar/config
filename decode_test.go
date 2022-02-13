package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func Test_DecodeFileSimpleConfig(t *testing.T) {
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

	if err := DecodeFile(&cfg, filepath.Join("testdata", "server.conf"), ErrorHandler(errh(t))); err != nil {
		t.Fatal(err)
	}
}

func Test_DecodeFileArrays(t *testing.T) {
	type Block struct {
		String string
	}

	var cfg struct {
		Strings   []string
		Ints      []int64
		Floats    []float64
		Bools     []bool
		Durations []time.Duration
		Sizes     []int64
		Blocks    []Block
	}

	if err := DecodeFile(&cfg, filepath.Join("testdata", "array.conf"), ErrorHandler(errh(t))); err != nil {
		t.Fatal(err)
	}

	Strings := []string{"one", "two", "three", "four", `"five"`}
	Ints := []int64{1, 2, 3, 4}
	Floats := []float64{1.2, 3.4, 5.6, 7.8}
	Bools := []bool{true, false}
	Durations := []time.Duration{time.Second, time.Minute * 2, time.Hour * 3}
	Sizes := []int64{1, 2048, 3145728, 4294967296, 5497558138880}
	Blocks := []Block{{"foo"}, {"bar"}, {"baz"}}

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

	for i, block := range Blocks {
		if cfg.Blocks[i].String != block.String {
			t.Errorf("Blocks[%d] - unexpected string, expected=%q, got=%q\n", i, block.String, cfg.Blocks[i].String)
		}
	}
}

func Test_DecodeNoGroupLabel(t *testing.T) {
	var cfg struct {
		Driver struct {
			SSH struct {
				Addr string

				Auth struct {
					Username string
					Identity string
				}
			}

			Docker struct {
				Host    string
				Version string
			}

			QEMU struct {
				Disks  string
				CPUs   int64
				Memory int64
			}
		} `config:",nogroup"`
	}

	if err := DecodeFile(&cfg, filepath.Join("testdata", "nogroup.conf"), ErrorHandler(errh(t))); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg.Driver)
}

func Test_DecodeFileLabel(t *testing.T) {
	type TLS struct {
		CA string
	}

	type Auth struct {
		Addr string

		TLS TLS
	}

	var cfg struct {
		Auth map[string]Auth

		Ports map[string][]string
	}

	if err := DecodeFile(&cfg, filepath.Join("testdata", "label.conf"), ErrorHandler(errh(t))); err != nil {
		t.Fatal(err)
	}

	expectedAuth := map[string]Auth{
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

	for label, auth := range expectedAuth {
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

	expectedPorts := map[string][]string{
		"open":  {"8080", "8443"},
		"close": {"80", "443"},
	}

	for label, ports := range expectedPorts {
		cfg, ok := cfg.Ports[label]

		if !ok {
			t.Fatalf("could not find label %q\n", label)
		}

		for i := range cfg {
			if ports[i] != cfg[i] {
				t.Fatalf("unxepected ports[%d], expected=%q, got=%q\n", i, ports[i], cfg[i])
			}
		}
	}
}

func Test_DecodeFileUTF8(t *testing.T) {
	var cfg struct {
		Block map[string]struct {
			Strings []string
		}
	}

	if err := DecodeFile(&cfg, filepath.Join("testdata", "utf8.conf"), ErrorHandler(errh(t))); err != nil {
		t.Fatal(err)
	}

	label := "标签"

	block, ok := cfg.Block[label]

	if !ok {
		t.Fatalf("could not find label %q\n", label)
	}

	expected := "细绳"

	for i, s := range block.Strings {
		if s != expected {
			t.Fatalf("cfg.Block[%q].Strings[%d] - unexpected string, expected=%q, got=%q\n", label, i, expected, s)
		}
	}
}

func Test_DecodeFileDuration(t *testing.T) {
	var cfg struct {
		Hour            time.Duration
		HourHalf        time.Duration `config:"hour_half"`
		HourHalfSeconds time.Duration `config:"hour_half_seconds"`
	}

	if err := DecodeFile(&cfg, filepath.Join("testdata", "duration.conf"), ErrorHandler(errh(t))); err != nil {
		t.Fatal(err)
	}
}

func Test_DecodeInclude(t *testing.T) {
	var cfg struct {
		Block map[string]struct {
			Strings []string
		}

		Hour            time.Duration
		HourHalf        time.Duration `config:"hour_half"`
		HourHalfSeconds time.Duration `config:"hour_half_seconds"`
	}

	opts := []Option{
		ErrorHandler(errh(t)),
		Includes,
	}

	if err := DecodeFile(&cfg, filepath.Join("testdata", "include.conf"), opts...); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg)
}

func Test_DecodeEnvVars(t *testing.T) {
	var cfg struct {
		Database struct {
			Addr     string
			Password string
		}
	}

	os.Setenv("DB_PASSWORD", "secret")

	opts := []Option{
		ErrorHandler(errh(t)),
		Envvars,
	}

	if err := DecodeFile(&cfg, filepath.Join("testdata", "envvars.conf"), opts...); err != nil {
		t.Fatal(err)
	}

	if cfg.Database.Password != "secret" {
		t.Fatalf("unexpected Database.Password, expected=%q, got=%q\n", "secret", cfg.Database.Password)
	}
}

func Test_DecodeDeprecated(t *testing.T) {
	var cfg struct {
		TLS struct {
			CA string
		}

		SSL struct {
			CA string
		} `config:",deprecated:tls"`
	}

	errs := make([]string, 0, 1)

	errh := func(pos Pos, msg string) {
		errs = append(errs, msg)
	}

	if err := DecodeFile(&cfg, filepath.Join("testdata", "deprecated.conf"), ErrorHandler(errh)); err != nil {
		t.Fatalf("expected decode to fail, it did not")
	}

	if errs[0] != "SSL is deprecated use tls instead" {
		t.Fatalf("could not find deprecated message")
	}
}
