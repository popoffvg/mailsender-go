package mail

import "github.com/caarlos0/env"

type MailConfig struct {
	Email       string `env:"MAIL_EMAIL"`
	Description string `env:"MAIL_DESCRIPTION"`

	Host     string `env:"MAIL_HOST"`
	Port     int    `env:"MAIL_PORT"`
	Login    string `env:"MAIL_LOGIN"`
	Password string `env:"MAIL_PASSWORD"`
}

func ReadConfig() (*MailConfig, error) {
	var cfg MailConfig
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
