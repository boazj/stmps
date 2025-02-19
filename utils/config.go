package utils

import (
	"github.com/spezifisch/stmps/consts"
	"github.com/spezifisch/stmps/logger"
	"github.com/spf13/viper"
)

type ConfigProvider interface {
	Log() logger.Logger
	Conf() *Config
}

type Config struct {
	Username      string
	Password      string
	PlaintextAuth bool

	Authentik bool
	ClientId  string
	AuthURL   string

	Host     string
	Scrobble bool

	RandomSongNumber uint

	Spinner string

	PlayerOptions map[string]string

	ClientName    string
	ClientVersion string
}

type ConfigProviderImpl struct {
	logger logger.Logger
	config *Config
}

func InitConfig() *Config {
	conf := Config{
		ClientName:    consts.ClientName,
		ClientVersion: consts.ClientVersion,
	}
	conf.Username = viper.GetString("auth.username")
	conf.Password = viper.GetString("auth.password")
	conf.Authentik = viper.GetBool("sso.authentik")
	conf.ClientId = viper.GetString("sso.clientid")
	conf.AuthURL = viper.GetString("sso.authurl")
	conf.Host = viper.GetString("server.host")
	conf.PlaintextAuth = viper.GetBool("auth.plaintext")
	conf.Scrobble = viper.GetBool("server.scrobble")
	conf.RandomSongNumber = viper.GetUint("client.random-songs")

	externalPlayerOptions := viper.Sub("mpv")
	playerOptions := make(map[string]string)
	playerOptions["audio-display"] = "no"
	playerOptions["video"] = "no"
	playerOptions["terminal"] = "no"
	playerOptions["demuxer-max-bytes"] = "30MiB"
	playerOptions["audio-client-name"] = "stmp"

	if externalPlayerOptions != nil {
		opts := externalPlayerOptions.AllSettings()
		for opt, value := range opts {
			playerOptions[opt] = value.(string)
		}
	}
	conf.PlayerOptions = playerOptions

	return &conf
}

func InitConfigProvider() *ConfigProviderImpl {
	conf := InitConfig()
	rawLogger := logger.Init(logger.Info)
	var l logger.Logger = &rawLogger
	return &ConfigProviderImpl{
		l,
		conf,
	}
}

func (c *ConfigProviderImpl) Log() logger.Logger {
	return c.logger
}

func (c *ConfigProviderImpl) Conf() *Config {
	return c.config
}
