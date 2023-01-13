package cmd

import (
	"testing"
)

func TestConfig(t *testing.T) {
	c := cliConfig.NewClient()
	expected := "http://localhost:9600"
	if c.Url != "http://localhost:9600" {
		t.Error("\nActual: ", c.Url, "\nExpected: ", expected)
	}
}
