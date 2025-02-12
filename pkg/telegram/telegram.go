package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"waterTelegram/config"
	"waterTelegram/pkg/database"
	"waterTelegram/pkg/post"
	"waterTelegram/pkg/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var userAction = make(map[int64]string)

func InitTelegramBot(config config.Config) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		return nil, err
	}
	bot.Debug = config.IsBotDebug
	log.Printf("Authorized on account %s", bot.Self.UserName)
	return bot, nil
}

func ProcessMessage(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, update.Message)
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update.CallbackQuery)
		}
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	chatID := callbackQuery.Message.Chat.ID
	switch callbackQuery.Data {
	case "deleteAll":
		err := database.DeleteSubscription(chatID)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "❌ Ошибка при отписке")
			bot.Send(msg)
			return
		}
		msg := tgbotapi.NewMessage(chatID, "✅ Вы успешно отписались от рассылки")
		bot.Send(msg)

	case "deleteMany":
		userAction[chatID] = "deleteMany"
		msg := tgbotapi.NewMessage(chatID, "Введите адреса для удаления, разделенные запятой.\nПример: 'Куйбышева 8, Ленина'")
		bot.Send(msg)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	switch message.Text {
	case "/start":
		msg := tgbotapi.NewMessage(chatID, "Привет! Я - бот для оповещения об отключении водоснабжения в Тамбове. Введите адрес для подписки. Пример: Куйбышева 8")
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Подписаться"),
				tgbotapi.NewKeyboardButton("Отписаться"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Мои подписки"),
			),
		)
		msg.ReplyMarkup = keyboard
		bot.Send(msg)

	case "Подписаться":
		userAction[chatID] = "subscribe"
		msg := tgbotapi.NewMessage(chatID, "Введите адрес для подписки. Пример: Куйбышева 8")
		bot.Send(msg)

	case "Отписаться":
		msg := tgbotapi.NewMessage(chatID, "Выберите: Удалить все подписки или выборочно")

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Удалить все подписки", "deleteAll"),
				tgbotapi.NewInlineKeyboardButtonData("Удалить несколько подписок", "deleteMany"),
			),
		)
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
		break

	case "Мои подписки":
		subs, err := database.GetAllSubscriptionsByChatID(chatID)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка при получении списка адресов пользователя"))
			return
		}

		if len(subs) == 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "ℹ️ У вас нет активных подписок."))
			return
		}

		var msgText strings.Builder
		msgText.WriteString("📋 Ваши подписки:\n\n")

		for i, sub := range subs {
			msgText.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, sub.Street, sub.Number))
		}
		bot.Send(tgbotapi.NewMessage(chatID, msgText.String()))

	case "/help":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Раздел в разработке..."))
	case "/admin":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "https://www.youtube.com/watch?v=zXz0InOf-z0"))
	default:
		if action, exists := userAction[chatID]; exists {
			switch action {
			case "subscribe":
				street, number := post.ParseAddress(message.Text)
				if street == "" {
					bot.Send(tgbotapi.NewMessage(chatID, "❌ Введите корректный адрес"))
					return
				}
				delete(userAction, chatID)
				lastPostID := GetLastPostID()
				err := database.SaveAddress(chatID, street, number, lastPostID)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка при сохранении адреса"))
					return
				}
				bot.Send(tgbotapi.NewMessage(chatID, "✅ Адрес успешно сохранен. Вы будете получать уведомления о новых постах."))
				return

			case "deleteMany":
				addresses := strings.Split(message.Text, ",")
				var deletedCount int
				for _, addr := range addresses {
					addr = strings.TrimSpace(addr)
					if addr == "" {
						continue
					}
					street, number := post.ParseAddress(addr)
					if street == "" {
						continue
					}
					err := database.DeleteManySubscription(chatID, street, number)
					if err != nil {
						log.Printf("Ошибка удаления записи %q: %v", addr, err)
						continue
					}
					deletedCount++
				}
				delete(userAction, chatID)
				if deletedCount > 0 {
					bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Успешно удалено %d записей", deletedCount)))
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, "❌ Не удалось удалить ни одной записи. Проверьте введенные адреса."))
				}
				return
			}

		}

		street, number := post.ParseAddress(message.Text)
		if street == "" {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Введите корректный адрес"))
		}

		posts := repository.GetPosts()
		if posts == nil {
			posts = post.GetPostsFromAPI()
			repository.UpdateCache(posts)
		}

		foundPosts := post.FindPostsByData(posts, street, number)
		if len(foundPosts) == 0 {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Адрес не найден"))
		} else {
			for _, post := range foundPosts {
				msgText := fmt.Sprintf("%s\nДата публикации: %s", post.Text, post.Date.Local().Format("02.01.2006 15:04"))
				bot.Send(tgbotapi.NewMessage(chatID, msgText))
			}
		}
	}
}

type SubNotification struct {
	ChatID   int64
	NewPosts []post.Post
}

func CheckNewPostsForSubs() ([]SubNotification, error) {
	subscriptions, err := database.GetAllSubscriptions()
	if err != nil {
		log.Println("Ошибка при получении подписок:", err)
		return nil, err
	}

	posts := repository.GetPosts()
	if posts == nil {
		posts = post.GetPostsFromAPI()
		repository.UpdateCache(posts)
	}

	var notifications []SubNotification
	for _, sub := range subscriptions {
		var newPosts []post.Post
		for _, p := range posts {
			if int64(p.Id) > sub.LastPostID {
				matchesStreet := sub.Street == "" || strings.Contains(strings.ToLower(p.Text), strings.ToLower(sub.Street))
				matchesNumber := sub.Number == "" || strings.Contains(strings.ToLower(p.Text), strings.ToLower(sub.Number))
				if matchesStreet && matchesNumber {
					newPosts = append(newPosts, p)
				}
			}
		}

		if len(newPosts) > 0 {
			notifications = append(notifications, SubNotification{ChatID: sub.ChatID, NewPosts: newPosts})
		}
	}

	return notifications, nil
}

func NotifySubs(bot *tgbotapi.BotAPI, notifications []SubNotification) {
	for _, notification := range notifications {
		for _, post := range notification.NewPosts {
			msgText := fmt.Sprintf("%s\nДата публикации: %s", post.Text, post.Date.Local().Format("02.01.2006 15:04"))
			msg := tgbotapi.NewMessage(notification.ChatID, msgText)
			bot.Send(msg)
		}

		lastPostID := notification.NewPosts[len(notification.NewPosts)-1].Id
		if err := database.UpdateLastPostID(notification.ChatID, int64(lastPostID)); err != nil {
			log.Println("Ошибка при обновлении ID последнего поста:", err)
		}
	}
}

func SendNotificationsSubs(bot *tgbotapi.BotAPI) {
	notifications, err := CheckNewPostsForSubs()
	if err != nil {
		log.Println("Ошибка при проверке новых постов:", err)
		return
	}

	if len(notifications) > 0 {
		NotifySubs(bot, notifications)
	}
}

func GetLastPostID() string {
	posts := repository.GetPosts()
	if posts == nil {
		posts = post.GetPostsFromAPI()
		repository.UpdateCache(posts)
	}

	if len(posts) == 0 {
		return ""
	}

	return strconv.Itoa(int(posts[0].Id))
}
