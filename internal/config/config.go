package config

import "github.com/caarlos0/env/v11"

type Config struct {
	GithubPAT string `env:"TTV_GITHUB_PAT,required"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
