package main

import (
	"encoding/json"
	"github.com/invopop/jsonschema"
	"os"
	"userservice/internal/config"
)

const path = "config/appconfig.schema.json"

func main() {

	s := jsonschema.Reflect(&config.AppConfig{})
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(path, []byte(data), 0644)
	if err != nil {
		panic(err)
	}
}
