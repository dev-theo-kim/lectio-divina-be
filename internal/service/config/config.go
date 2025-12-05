// Package config provides the configuration for the service.
package config

type Config struct {
	Server struct {
		Swagger bool `toml:"swagger"`
		Major   int  `toml:"major"`
		Minor   int  `toml:"minor"`
		Patch   int  `toml:"patch"`
	} `toml:"server"`

	MySQL map[string]MySQL `toml:"mysql"`

	Log map[string]Log `toml:"log"`
}

type MySQL struct {
	DB   string `toml:"db"`
	Host string `toml:"host"`
	User string `toml:"user"`
	Pass string `toml:"pass"`
}

type Log struct {
	Use   bool   `toml:"use"`
	Level string `toml:"level"`
	Path  string `toml:"path"`
}
