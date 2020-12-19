package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"

	eztunnel "github.com/penguinpowernz/eztunnel/pkg"
	"golang.org/x/crypto/ssh"
)

func generateConfigIfNeeded(cfgFile string) (err error) {
	if _, err = os.Stat(cfgFile); !os.IsNotExist(err) {
		return
	}

	fmt.Println("First run, generating new config")

	cfg := eztunnel.Config{ListenPort: ":8022", Tunnels: []eztunnel.Tunnel{}}

	_, cfg.HostKey, err = MakeSSHKeyPair()
	if err != nil {
		return
	}

	cfg.PublicKey, cfg.PrivateKey, err = MakeSSHKeyPair()
	if err != nil {
		return
	}

	cfg.Filename = cfgFile
	return cfg.Save()
}

func clear() {
	cmd := exec.Command("/usr/bin/clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func MakeSSHKeyPair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", "", err
	}

	// generate and write private key as PEM
	var privKeyBuf bytes.Buffer

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
		return "", "", err
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	var pubKeyBuf bytes.Buffer
	pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))

	user := os.Getenv("USER")
	hn, _ := os.Hostname()
	pubKeyBuf.Truncate(pubKeyBuf.Len() - 1)
	pubKeyBuf.WriteString(" " + user + "@" + hn + "\n")

	return pubKeyBuf.String(), privKeyBuf.String(), nil
}
