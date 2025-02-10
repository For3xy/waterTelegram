package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	TelegramToken    string  `json:"telegram_token"`
	IsBotDebug       bool    `json:"is_bot_debug"`
	SrvAccessKey     string  `json:"srv_access_key"`
	Version          float64 `json:"version"`
	Domain           string  `json:"domain"`
	AutosaveInterval int     `json:"autosave_interval"`
}

func LoadConfig() Config {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Fatal(err)
	}

	return config
}
