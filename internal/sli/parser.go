package sli

import "errors"

var (
    ErrInvalidPathArg = errors.New(`path for saving the file is not specified`)
    ErrInvalidLinkArg = errors.New(`download urls are not set`)
)

// Содержит информацию о конфигурации, которая была передана в аргументах CLI
type Config struct {
	SavePath string
	URLs []string
}

func ParseArgs(args []string) (*Config, error) {
	if len(args) < 2 {
		return nil, ErrInvalidPathArg
	}
	if len(args) < 3 {
		return nil, ErrInvalidLinkArg
	}

	return &Config{
		SavePath: args[1],
		URLs: args[2:],
	}, nil
}

