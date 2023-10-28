package cmd

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/NETWAYS/check_logstash/internal/client"
	"github.com/NETWAYS/go-check"
	checkhttpconfig "github.com/NETWAYS/go-check-network/http/config"
)

type Config struct {
	BasicAuth string `env:"CHECK_LOGSTASH_BASICAUTH"`
	Bearer    string `env:"CHECK_LOGSTASH_BEARER"`
	CAFile    string `env:"CHECK_LOGSTASH_CA_FILE"`
	CertFile  string `env:"CHECK_LOGSTASH_CERT_FILE"`
	KeyFile   string `env:"CHECK_LOGSTASH_KEY_FILE"`
	Hostname  string `env:"CHECK_LOGSTASH_HOSTNAME"`
	Port      int
	Info      bool
	Insecure  bool
	PReady    bool
	Secure    bool
}

const Copyright = `
Copyright (C) 2022 NETWAYS GmbH <info@netways.de>
`

const License = `
Copyright (C) 2022 NETWAYS GmbH <info@netways.de>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see https://www.gnu.org/licenses/.
`

var (
	cliConfig Config
)

func (c *Config) NewClient() *client.Client {
	u := url.URL{
		Scheme: "http",
		Host:   c.Hostname + ":" + strconv.Itoa(c.Port),
	}

	if c.Secure {
		u.Scheme = "https"
	}

	// Create TLS configuration for default RoundTripper
	tlsConfig, err := checkhttpconfig.NewTLSConfig(&checkhttpconfig.TLSConfig{
		InsecureSkipVerify: c.Insecure,
		CAFile:             c.CAFile,
		KeyFile:            c.KeyFile,
		CertFile:           c.CertFile,
	})

	if err != nil {
		check.ExitError(err)
	}

	var rt http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
	}

	// Using a Bearer Token for authentication
	if c.Bearer != "" {
		rt = checkhttpconfig.NewAuthorizationCredentialsRoundTripper("Bearer", c.Bearer, rt)
	}

	// Using a BasicAuth for authentication
	if c.BasicAuth != "" {
		s := strings.Split(c.BasicAuth, ":")
		if len(s) != 2 {
			check.ExitError(fmt.Errorf("specify the user name and password for server authentication <user:password>"))
		}

		var u = s[0]

		var p = s[1]

		rt = checkhttpconfig.NewBasicAuthRoundTripper(u, p, rt)
	}

	return client.NewClient(u.String(), rt)
}
