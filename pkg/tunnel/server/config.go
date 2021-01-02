package server

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/gliderlabs/ssh"
	"github.com/penguinpowernz/mole/internal/util"
	gossh "golang.org/x/crypto/ssh"
)

// Config is a server config
type Config struct {
	Filename       string   `json:"-"`
	AuthorizedKeys []string `json:"authorized_keys"`
	RunServer      bool     `json:"run_server"`
	ListenPort     string   `json:"listen_port"`
	HostKey        string   `json:"host_key"`
}

// AuthorizedKeyBytes will return the authorized keys as a byte array
func (cfg Config) AuthorizedKeyBytes() []byte {
	s := ""
	for _, k := range cfg.AuthorizedKeys {
		s += k + "\n"
	}
	return []byte(s)
}

func parseString(in []byte) (out, rest []byte, ok bool) {
	if len(in) < 4 {
		return
	}
	length := binary.BigEndian.Uint32(in)
	in = in[4:]
	if uint32(len(in)) < length {
		return
	}
	out = in[:length]
	rest = in[length:]
	ok = true
	return
}

// AddAuthorizedKey will add a new authorized key, can either be a string or an ssh.PublicKey object
func (cfg *Config) AddAuthorizedKey(key interface{}) {
	switch v := key.(type) {
	case string:
		cfg.AuthorizedKeys = append(cfg.AuthorizedKeys, v)
	case ssh.PublicKey:
		cfg.AuthorizedKeys = append(cfg.AuthorizedKeys, string(gossh.MarshalAuthorizedKey(v)))
	}
}

// Save will save the config file
func (cfg Config) Save() error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cfg.Filename, data, 0644)
}

// LoadConfig will load the config from the given filename
func LoadConfig(fn string) (cfg *Config, err error) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return
	}
	cfg = new(Config)
	cfg.Filename = fn
	err = yaml.Unmarshal(data, cfg)
	return
}

// GenerateConfig will generate a config with the host key preset
func GenerateConfig() Config {
	cfg := Config{ListenPort: ":8022", RunServer: true}

	var err error
	_, cfg.HostKey, err = util.MakeSSHKeyPair()
	if err != nil {
		return cfg
	}

	return cfg
}

// GenerateConfigIfNeeded will generate a new config to the given filename
// if it doesn't already exist
func GenerateConfigIfNeeded(cfgFile string) (err error) {
	if _, err = os.Stat(cfgFile); !os.IsNotExist(err) {
		return
	}

	fmt.Println("First run, generating new config")

	cfg := GenerateConfig()
	cfg.Filename = cfgFile
	return cfg.Save()
}
