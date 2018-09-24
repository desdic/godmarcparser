package cfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

// HTTPCfg hold the http configuration
type HTTPCfg struct {
	Port         string `json:"port"`
	WriteTimeout int    `json:"writeTimeout"`
	ReadTimeout  int    `json:"readTimeout"`
	IdleTimeout  int    `json:"idleTimeout"`
}

// StorageCfg hold the storage configuration
type StorageCfg struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// LogCfg hold the log configuration
type LogCfg struct {
	Level string `json:"level"`
}

// ScanDirectory hold the directory configuration
type ScanDirectory struct {
	Path     string `json:"path"`
	Interval int    `json:"interval"`
}

// Config hold the configuration for dmarc
type Config struct {
	HTTP      HTTPCfg       `json:"http"`
	Storage   StorageCfg    `json:"storage"`
	Log       LogCfg        `json:"log"`
	Directory ScanDirectory `json:"directory"`
}

func (c *Config) sanitize() {

	// HTTP
	if c.HTTP.Port == "" {
		c.HTTP.Port = ":8080"
	}
	if c.HTTP.WriteTimeout < 15 {
		c.HTTP.WriteTimeout = 15
	}
	if c.HTTP.ReadTimeout < 15 {
		c.HTTP.ReadTimeout = 15
	}
	if c.HTTP.IdleTimeout < 30 {
		c.HTTP.IdleTimeout = 30
	}

	// Directory
	if c.Directory.Interval < 30 {
		c.Directory.Interval = 30
	}
}

// ReadConfig reads a config file and returns the Config
func ReadConfig(path string) (Config, error) {

	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("Unable to open %s: %v", path, err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Errorf("Unable to close config file: %v", cerr)
		}
	}()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return Config{}, fmt.Errorf("Unable to read %s: %v", path, err)
	}

	var cfg Config
	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("Unable parse json: %v", err)
	}

	cfg.sanitize()

	return cfg, nil
}
