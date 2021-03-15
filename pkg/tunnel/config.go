package tunnel

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/penguinpowernz/mole/internal/util"
)

// Config represents the config file for the tunnel client
type Config struct {
	Filename string    `json:"-"`
	Tunnels  []Tunnel  `json:"tunnels"`
	Keys     []KeyPair `json:"keys"`
}

type KeyPair struct {
	Address string `json:"address"`
	Private string `json:"private"`
	Public  string `json:"public"`
}

func (cfg Config) KeyForAddress(addr string) string {
	def := ""
	for _, k := range cfg.Keys {
		if k.Address == "*" {
			def = k.Private
		}
		if k.Address == addr {
			return k.Private
		}
	}
	return def
}

func (cfg Config) DefaultKey() string {
	return cfg.KeyForAddress("*")
}

// Save will save the config to disk
func (cfg Config) Save() error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cfg.Filename, data, 0644)
}

// LoadConfig will load the config from disk
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

// GenerateConfig will generate a config with new private
// and public key
func GenerateConfig() Config {
	cfg := Config{}

	pub, priv, err := util.MakeSSHKeyPair()
	if err != nil {
		return cfg
	}
	cfg.Keys = append(cfg.Keys, KeyPair{Address: "*", Private: priv, Public: pub})

	return cfg
}

// GenerateConfigIfNeeded will only generate a new file if the
// given filename does not exist
func GenerateConfigIfNeeded(cfgFile string) (err error) {
	if _, err = os.Stat(cfgFile); !os.IsNotExist(err) {
		return
	}

	fmt.Println("First run, generating new config")

	cfg := GenerateConfig()
	cfg.Filename = cfgFile
	return cfg.Save()
}
