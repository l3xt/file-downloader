package cli

import (
	"fmt"
)

// Содержит информацию о конфигурации, которая была передана в аргументах CLI
type Config struct {
	SavePath string
	URLs     []string
}

func ParseArgs(args []string) (*Config, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("path for saving the file is not specified")
	}
	if len(args) < 3 {
		return nil, fmt.Errorf("download urls are not set")
	}

	return &Config{
		SavePath: args[1],
		URLs:     args[2:],
	}, nil
}
