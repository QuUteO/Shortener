package config

import (
	"errors"

	"github.com/wb-go/wbf/config/cleanenv-port"
)

type Config struct {
	Env      Env      `yaml:"env"`
	Postgres Postgres `yaml:"postgres"`
	Redis    Redis    `yaml:"redis"`
	HTTP     HTTP     `yaml:"http"`
}

type Env struct {
	Env string `env:"env" envDefault:"local"`
}

type Postgres struct {
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	Username    string `yaml:"user"`
	Password    string `yaml:"password"`
	Database    string `yaml:"database"`
	DatabaseDSN string `yaml:"database_dsn"`
}

type Redis struct {
	Addr     string  `yaml:"address"`
	Password string  `yaml:"password"`
	Attempts int     `yaml:"attempts"`
	Backoff  float64 `yaml:"backoff"`
}

type HTTP struct {
	Addr string `yaml:"address"`
}

func Init(path string) (*Config, error) {
	var cfg Config

	if err := cleanenvport.LoadPath(path, &cfg); err != nil {
		return nil, errors.New("Ошибка загрузки конфига:" + err.Error())
	}

	return &cfg, nil
}
