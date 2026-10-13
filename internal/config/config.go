package config

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

type Configuration struct {
	ConfigPairs []ConfigPair
}

type ConfigPair struct {
	Modifier string
	Key      string

	Action   string
	Argument string
}

func ReadConfig() error {
	var data map[string]string

	var config Configuration

	_, err := toml.DecodeFile("config.toml", &data)
	if err != nil {
		return fmt.Errorf("could not decode file: %w", err)
	}

	for key, value := range data {
		configPair, err := extractConfigPair(key, value)
		if err != nil {
			return err
		}

		config.ConfigPairs = append(config.ConfigPairs, *configPair)
	}

	return nil
}

func extractConfigPair(key, value string) (*ConfigPair, error) {
	keys := strings.Split(key, "+")
	if len(keys) != 2 {
		return nil, fmt.Errorf("invalid configuration")
	}

	values := strings.Split(value, ":")
	if len(values) == 2 {
		return nil, fmt.Errorf("invalid configuration")
	}

	return &ConfigPair{
		Modifier: keys[0],
		Key:      keys[1],

		Action:   values[0],
		Argument: values[1],
	}, nil
}
