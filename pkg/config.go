package eztunnel

import (
	"encoding/binary"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type Config struct {
	Filename       string   `json:"-"`
	AuthorizedKeys []string `json:"authorized_keys"`
	RunServer      bool     `json:"run_server"`
	Connect        bool     `json:"connect"`
	ListenPort     string   `json:"listen_port"`
	Tunnels        []Tunnel `json:"tunnels"`
	HostKey        string   `json:"host_key"`
	PublicKey      string   `json:"public_key"`
	PrivateKey     string   `json:"private_key"`
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
