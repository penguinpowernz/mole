package util

import "os"

// ConfigFiles is an ordered list of config files to search for
var ConfigFiles = []string{
	"./mole.yml",
	"~/.mole.yml",
	"~/.config/mole.yml",
	"~/.local/mole/mole.yml",
	"/etc/mole.yml",
}

// FindConfig will try to find the config file automatically and
// will return the filename that it does find, in the following
// order:
//
// - ./mole.yml
// - ~/.mole.yml
// - ~/.config/mole.yml
// - ~/.local/mole/mole.yml
// - /etc/mole.yml
//
func FindConfig() (string, bool) {
	for _, fn := range ConfigFiles {
		if _, err := os.Stat(fn); err == nil {
			return fn, true
		}
	}

	return "", false
}
