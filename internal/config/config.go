package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Configuration struct {
	ConfigPairs []ConfigPair
}

type ConfigPair struct {
	Modifiers uint32
	Keysym    uint32

	Action   string
	Argument string
}

func ReadConfig(path string) (*Configuration, error) {
	var data map[string]string

	var config Configuration

	configPath := path

	if configPath == "" {
		baseConfigDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("could not get config dir: %w", err)
		}

		configPath = filepath.Join(baseConfigDir, "magnet", "config.toml")
	}

	if _, err := toml.DecodeFile(configPath, &data); err != nil {
		return nil, fmt.Errorf("could not decode file: %w", err)
	}

	for key, value := range data {
		configPair, err := extractConfigPair(key, value)
		if err != nil {
			return nil, err
		}

		config.ConfigPairs = append(config.ConfigPairs, *configPair)
	}

	return &config, nil
}

func extractConfigPair(key, value string) (*ConfigPair, error) {
	keys := strings.Split(key, "+")
	if len(keys) < 2 {
		return nil, fmt.Errorf("invalid binding %q: expected Modifier+Key", key)
	}

	keyName := keys[len(keys)-1]

	keysym := KeysymFromName(keyName)
	if keysym == 0 {
		return nil, fmt.Errorf("invalid binding %q: unknown keysym %q", key, keyName)
	}

	var mods uint32

	for _, modName := range keys[:len(keys)-1] {
		name, err := ModifierFromName(modName)
		if err != nil {
			return nil, fmt.Errorf("invalid binding %q: %w", key, err)
		}

		mods |= name
	}

	values := strings.SplitN(value, ":", 2)
	if len(values) != 2 || values[0] == "" || values[1] == "" {
		return nil, fmt.Errorf("invalid action %q: expected action:argument", value)
	}

	return &ConfigPair{
		Modifiers: mods,
		Keysym:    keysym,

		Action:   values[0],
		Argument: values[1],
	}, nil
}
