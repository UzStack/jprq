package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	DomainName          string
	EventServerHost     string
	PublicServerHost    string
	PublicServerTLSHost string
	MaxTunnelsPerUser   int
	MaxConsPerTunnel    int
	EventServerPort     uint16
	PublicServerPort    uint16
	PublicServerTLSPort uint16
	TLSCertFile         string
	TLSKeyFile          string
	GithubClientID      string
	GithubClientSecret  string
	AllowedUsersFile    string
	AuthToken           string
	TLSDisabled         bool
}

func envPort(name string, fallback uint16) (uint16, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	port, err := strconv.ParseUint(value, 10, 16)
	if err != nil || port == 0 {
		return 0, errors.New(name + " must be a valid TCP port")
	}
	return uint16(port), nil
}

func (c *Config) Load() error {
	c.MaxTunnelsPerUser = 4
	c.MaxConsPerTunnel = 24
	var err error
	c.PublicServerPort, err = envPort("JPRQ_HTTP_PORT", 80)
	if err != nil {
		return err
	}
	c.EventServerPort, err = envPort("JPRQ_EVENT_PORT", 4321)
	if err != nil {
		return err
	}
	c.PublicServerTLSPort, err = envPort("JPRQ_HTTPS_PORT", 443)
	if err != nil {
		return err
	}
	c.DomainName = os.Getenv("JPRQ_DOMAIN")
	c.EventServerHost = os.Getenv("JPRQ_EVENT_HOST")
	c.PublicServerHost = os.Getenv("JPRQ_HTTP_HOST")
	c.PublicServerTLSHost = os.Getenv("JPRQ_HTTPS_HOST")
	c.TLSKeyFile = os.Getenv("JPRQ_TLS_KEY")
	c.TLSCertFile = os.Getenv("JPRQ_TLS_CERT")
	c.GithubClientID = os.Getenv("GITHUB_CLIENT_ID")
	c.GithubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	c.AuthToken = os.Getenv("JPRQ_AUTH_TOKEN")
	c.TLSDisabled = os.Getenv("JPRQ_TLS_DISABLED") == "1"
	c.AllowedUsersFile = "/etc/jprq/allowed-users.csv"

	if c.DomainName == "" {
		return errors.New("jprq domain env is not set")
	}
	if !c.TLSDisabled && (c.TLSKeyFile == "" || c.TLSCertFile == "") {
		return errors.New("TLS key/cert file is missing")
	}
	if c.AuthToken == "" && (c.GithubClientID == "" || c.GithubClientSecret == "") {
		return errors.New("github client id/secret is missing")
	}
	return nil
}
