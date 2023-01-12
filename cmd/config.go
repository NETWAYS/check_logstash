package cmd

import (
	"fmt"
	"github.com/NETWAYS/check_logstash/internal/client"
	"github.com/NETWAYS/check_logstash/internal/config"
	"github.com/NETWAYS/go-check"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BasicAuth string
	Bearer    string
	CAFile    string
	CertFile  string
	KeyFile   string
	Hostname  string
	Info      bool
	Insecure  bool
	PReady    bool
	Port      int
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
	tlsConfig, err := config.NewTLSConfig(&config.TLSConfig{
		InsecureSkipVerify: c.Insecure,
		CAFile:             c.CAFile,
		KeyFile:            c.KeyFile,
		CertFile:           c.CertFile,
	})

	if err != nil {
		check.ExitError(err)
	}

	var rt http.RoundTripper = &http.Transport{
		TLSClientConfig:       tlsConfig,
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
	}

	// Using a Bearer Token for authentication
	if c.Bearer != "" {
		var t config.Secret = config.Secret(c.Bearer)
		rt = config.NewAuthorizationCredentialsRoundTripper("Bearer", t, rt)
	}

	// Using a BasicAuth for authentication
	if c.BasicAuth != "" {
		s := strings.Split(c.BasicAuth, ":")
		if len(s) != 2 {
			check.ExitError(fmt.Errorf("Specify the user name and password for server authentication <user:password>"))
		}

		var u string = s[0]

		var p config.Secret = config.Secret(s[1])

		rt = config.NewBasicAuthRoundTripper(u, p, "", rt)
	}

	return client.NewClient(u.String(), rt)
}