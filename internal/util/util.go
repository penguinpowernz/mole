package util

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/crypto/ssh"
)

func Clear() {
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

func PrintLogsUntilEnter() {
	fmt.Println("Push enter to show the menu, otherwise logs will be printed")
	WaitForEnter()
}

func WaitForEnter() {
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
