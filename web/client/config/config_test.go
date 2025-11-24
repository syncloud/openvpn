package config_test

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	c := config.New()
	assert.Equal(t, c.Auth, "SHA256")
}

func TestTemplateGeneration(t *testing.T) {
	c := New()
	txt, err := ioutil.ReadFile("./templates/client-config.tpl")
	assert.Nil(t, err)

	_, err = GetText(string(txt), c)
	assert.Nil(t, err)
}

func TestBrokenTemplate(t *testing.T) {
	c := New()

	_, err := GetText("{{ ", c)
	assert.NotNil(t, err, "Parser should fail on broken template")
}
