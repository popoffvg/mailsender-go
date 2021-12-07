package config

import (
	"time"

	"github.com/caarlos0/env"
	"mts.teta.mailsender/internal/mail"
)

type Config struct {
	mail.MailConfig
	Addr string `env:"MAILSENDER_ADDR" envDefault:"0.0.0.0:8084"` // address of profile service

	Mongo   MongoConfig
}

type MongoConfig struct {
	Addr    string        `env:"MONGO_HOST" envDefault:""`
	Port    string        `env:"MONGO_PORT" envDefault:"27017"`
	User    string        `env:"MONGO_USER" envDefault:""`
	Pwd     string        `env:"MONGO_PASSWORD" envDefault:""`
	DB      string        `env:"MONGO_DB" envDefault:"profile"`
	Auth    string        `env:"MONGO_AUTH"`
	Timeout time.Duration `env:"MONGO_TIMEOUT" envDefault:"5s"`
}


func FromEnv() (*Config, error) {
	config := Config{}
	mongo := MongoConfig{}

	err := env.Parse(&mongo)
	if err != nil {
		return nil, err
	}
	config.Mongo = mongo

	err = env.Parse(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
