package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Defaults Default
	Commands map[string]string
	Hosts    map[string]*Host
}

type Default struct {
	Pool     string
	Surelog  *string
	Rsynclog *string
}

type Host struct {
	Mirror *string
	Fs     []*FileSystem
}

type FileSystem struct {
	Vg     *string
	Volume string
	Base   string
	Clean  *string
	Style  string
}

func LoadConfig(path string) (config *Config, err error) {
	var conf Config
	_, err = toml.DecodeFile(path, &conf)
	if err != nil {
		return
	}

	config = &conf
	return
}
