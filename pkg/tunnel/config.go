package tunnel

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/penguinpowernz/mole/internal/util"
)

type Config struct {
	Filename   string   `json:"-"`
	Tunnels    []Tunnel `json:"tunnels"`
	PublicKey  string   `json:"public_key"`
	PrivateKey string   `json:"private_key"`
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
	cfg := Config{}

	var err error
	cfg.PublicKey, cfg.PrivateKey, err = util.MakeSSHKeyPair()
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
