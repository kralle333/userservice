package main

import (
	"flag"
	"fmt"
	"userservice/internal/application"
	"userservice/internal/config"
)

var (
	configPath = flag.String("config", "config/debug.json", "Path to config file")
)

func main() {
	flag.Parse()

	if configPath == nil || *configPath == "" {
		panic("Config file path is required")
	}
	cfg, err := config.ReadConfig(*configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read config: %v", err))
	}

	app := application.NewApp(cfg)
	err = app.Run()
	if err != nil {
		panic(err)
	}
}
