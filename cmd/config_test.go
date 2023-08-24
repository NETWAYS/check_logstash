package cmd

import (
	"testing"
)

func TestConfig(t *testing.T) {
	c := cliConfig.NewClient()
	expected := "http://localhost:9600"
	if c.URL != "http://localhost:9600" {
		t.Error("\nActual: ", c.URL, "\nExpected: ", expected)
	}
}
