package server

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/gliderlabs/ssh"
	"github.com/penguinpowernz/eztunnel/internal/util"
	gossh "golang.org/x/crypto/ssh"
)

type Config struct {
	Filename       string   `json:"-"`
	AuthorizedKeys []string `json:"authorized_keys"`
	RunServer      bool     `json:"run_server"`
	ListenPort     string   `json:"listen_port"`
	HostKey        string   `json:"host_key"`
}

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

func (cfg *Config) AddAuthorizedKey(key interface{}) {
	switch v := key.(type) {
	case string:
		cfg.AuthorizedKeys = append(cfg.AuthorizedKeys, v)
	case ssh.PublicKey:
		cfg.AuthorizedKeys = append(cfg.AuthorizedKeys, string(gossh.MarshalAuthorizedKey(v)))
	}
}

func (cfg Config) Save() error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cfg.Filename, data, 0644)
}

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

func GenerateConfig() Config {
	cfg := Config{ListenPort: ":8022", RunServer: true}

	var err error
	_, cfg.HostKey, err = util.MakeSSHKeyPair()
	if err != nil {
		return cfg
	}

	return cfg
}

func GenerateConfigIfNeeded(cfgFile string) (err error) {
	if _, err = os.Stat(cfgFile); !os.IsNotExist(err) {
		return
	}

	fmt.Println("First run, generating new config")

	cfg := GenerateConfig()
	cfg.Filename = cfgFile
	return cfg.Save()
}
