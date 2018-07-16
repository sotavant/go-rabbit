package lib

import (
	"github.com/BurntSushi/toml"
)

type AppConfig struct {
	Rabbit rabbit
	Common common
	Database database
}

type rabbit struct {
	User string
	Pass string
	Host string
	QueueName string
}

type common struct {
	StreamCount int
	PathToPhp string
	PathToCreator string
	PathToPostScript string
	PathToPhpLog string
	PathToGoLog string
	PathToResultDoc string
	PathToResultZip string
	DocGenError string
	DocCodeSalt string
}

type database struct {
	*Db
	SuccessCode int
	ErrorCode int
	TaskIdField string
	LogField string
	ResultField string
	CodeField string
}

type Db struct {
	User string
	Pass string
	Database string
	Table string
}

func Config() (config AppConfig, err error) {
	_, err = toml.DecodeFile("config/config", &config)

	return config, err
}