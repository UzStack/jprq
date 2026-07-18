package config

import (
	"fmt"
	"testing"
)

func TestConfig_Load(t *testing.T) {
	t.Setenv("JPRQ_DOMAIN", "jprq.site")
	t.Setenv("JPRQ_TLS_KEY", "key.pem")
	t.Setenv("JPRQ_TLS_CERT", "cert.pem")
	t.Setenv("JPRQ_AUTH_TOKEN", "test-token")

	config := &Config{}
	err := config.Load()
	if err != nil {
		t.Logf("Error while loading the config: %v", err.Error())
		t.Fail()
	}
}

func TestConfigSelfHostedPorts(t *testing.T) {
	t.Setenv("JPRQ_DOMAIN", "jprq.example.com")
	t.Setenv("JPRQ_AUTH_TOKEN", "test-token")
	t.Setenv("JPRQ_TLS_DISABLED", "1")
	t.Setenv("JPRQ_HTTP_PORT", "18080")
	t.Setenv("JPRQ_EVENT_PORT", "14321")
	t.Setenv("JPRQ_HTTP_HOST", "127.0.0.1")

	var config Config
	if err := config.Load(); err != nil {
		t.Fatal(err)
	}
	if config.PublicServerPort != 18080 || config.EventServerPort != 14321 {
		t.Fatalf("unexpected ports: http=%d events=%d", config.PublicServerPort, config.EventServerPort)
	}
	if config.PublicServerHost != "127.0.0.1" {
		t.Fatalf("unexpected HTTP bind host: %q", config.PublicServerHost)
	}
}

func TestConfig_loadEmptyEnv(t *testing.T) {
	envs := []struct {
		key     string
		value   string
		ErrText string
	}{
		{
			"JPRQ_DOMAIN",
			"jprq.site",
			"jprq domain env is not set",
		},
		{
			"JPRQ_TLS_KEY",
			"example.key",
			"TLS key/cert file is missing",
		},
		{
			"JPRQ_TLS_CERT",
			"example.cert",
			"TLS key/cert file is missing",
		},
		{
			"GITHUB_CLIENT_ID",
			"client-id",
			"github client id/secret is missing",
		},
		{
			"GITHUB_CLIENT_SECRET",
			"client-secret",
			"github client id/secret is missing",
		},
	}

	for i, missing := range envs {
		t.Run(fmt.Sprintf("Missing %s", missing.key), func(t *testing.T) {
			for j, env := range envs {
				if i == j {
					continue
				}
				t.Setenv(env.key, env.value)
			}
			config := &Config{}
			err := config.Load()

			if err == nil {
				t.Logf("expected %v, but got %v", missing.ErrText, err)
			}
			if err.Error() != missing.ErrText {
				t.Logf("expected %s, but got %s", missing.ErrText, err.Error())
			}
		})
	}
}
