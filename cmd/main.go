package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"waterTelegram/config"
	"waterTelegram/pkg/database"
	"waterTelegram/pkg/scheduler"
	"waterTelegram/pkg/telegram"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cfg := config.LoadConfig()

	database.InitDB()

	bot, err := telegram.InitTelegramBot(cfg)
	if err != nil {
		log.Fatal("Ошибка при инициализации бота:", err)
		os.Exit(1)
	}

	fmt.Println("Бот успешно запущен")

	scheduler.StartScheduler()

	go telegram.ProcessMessage(bot)

	ticker := time.NewTicker(time.Duration(cfg.AutosaveInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			telegram.SendNotificationsSubs(bot)
		}
	}

}
