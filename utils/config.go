package utils

import "github.com/spezifisch/stmps/logger"

type ConfigProvider interface{} // TODO: implement

type Config struct{} // TODO: implement

type ConfigProviderImpl struct {
	logger logger.Logger
	config Config
}
