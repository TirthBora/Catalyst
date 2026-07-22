package config

import "time"

type Config struct {
	Server  ServerConfig
	Browser BrowserConfig
	Watcher WatcherConfig
}

type ServerConfig struct {
	Port int
}

type BrowserConfig struct {
	Open bool
}

type WatcherConfig struct {
	Debounce time.Duration
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Browser: BrowserConfig{
			Open: true,
		},
		Watcher: WatcherConfig{
			Debounce: 200 * time.Millisecond,
		},
	}
}
