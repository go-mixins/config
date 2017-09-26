package main

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/go-mixins/config"
)

// Config represent some configuration stored in JSON or YAML file
type Config struct {
	DbURI string
}

// App is an example application that live-reloads its configuration
type App struct {
	*Config

	mu        sync.RWMutex
	newConfig *Config
	reloader  *config.Config
}

// ReadConfig parses configuration file and optionally validates it
func (app *App) ReadConfig(r io.Reader) error {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return json.NewDecoder(r).Decode(&app.newConfig)
}

// ReplaceConfig replaces app configuration with new one
func (app *App) ReplaceConfig(err error) {
	if err != nil {
		// Config is invalid. Do nothing.
		return
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	app.Config = app.newConfig
	app.newConfig = nil
}

func main() {
	app := new(App)
	var err error
	if app.reloader, err = config.New("config.json", app.ReadConfig, app.ReplaceConfig); err != nil {
		panic(err)
	}
	// at this point app.Config is loaded and parsed
}
