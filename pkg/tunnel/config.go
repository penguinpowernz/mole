package tunnel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/penguinpowernz/mole/internal/util"
)

// Config represents the config file for the tunnel client
type Config struct {
	Filename string `json:"-"`
	Clients  []*Client
}

// UnmarshalJSON is used because the config file contains an array but
// our config object is a struct, we want to put that array into the
// Clients field of the struct
func (cfg *Config) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &cfg.Clients); err != nil {
		return err
	}

	cfg.copyDefaultKeys()

	return nil
}

func (cfg Config) copyDefaultKeys() {
	def := cfg.ClientWithAddress("*")
	if def == nil {
		return
	}
	if def.Private == "" && def.Public == "" {
		return
	}

	for _, cl := range cfg.Clients {
		if cl.Private == "" {
			cl.Private = def.Private
		}
		if cl.Public == "" {
			cl.Public = def.Public
		}
	}
}

// ClientWithAddress will return the client object for the given address
func (cfg Config) ClientWithAddress(addr string) *Client {
	for _, c := range cfg.Clients {
		if c.Address == addr {
			return c
		}
	}
	return nil
}

// KeyForAddress will return the keypair for the given address
func (cfg Config) KeyForAddress(addr string) string {
	var def string
	for _, c := range cfg.Clients {
		if c.Address == "*" {
			def = c.Private
		}
		if c.Address == addr {
			return c.Private
		}
	}
	return def
}

// DefaultKey will return the default private key that should
// be used for any tunnels that don't specify one
func (cfg Config) DefaultKey() string {
	return cfg.KeyForAddress("*")
}

// Tunnels will return the tunnels from all clients
func (cfg Config) Tunnels() Tunnels {
	tuns := []*Tunnel{}
	for _, c := range cfg.Clients {
		tuns = append(tuns, c.Tunnels...)
	}
	return tuns
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

	cfg.Clients = append(cfg.Clients, &Client{Address: "*", Private: priv, Public: pub})
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
