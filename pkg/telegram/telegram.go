package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
	"waterTelegram/config"
	"waterTelegram/pkg/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Post struct {
	Id   float64   `json:"id"`
	Text string    `json:"text"`
	Date time.Time `json:"date"`
	// можно добавить поле типа массива, куда при получении будут складываться названия улиц, например
}

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
	case "subscribe":
		userAction[chatID] = "subscribe"
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Введите адрес для подписки. Пример: Куйбышева 8")
		bot.Send(msg)

	case "unsubscribe":
		err := database.DeleteSubscription(callbackQuery.Message.Chat.ID)
		if err != nil {
			msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Ошибка при отписке")
			bot.Send(msg)
			return
		}
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "✅ Вы успешно отписались от рассылки")
		bot.Send(msg)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	switch message.Text {
	case "/start":
		msg := tgbotapi.NewMessage(message.Chat.ID, "Привет! Я - бот для оповещения об отключении водоснабжения в Тамбове. Введите адрес для подписки. Пример: Куйбышева 8")
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Подписаться", "subscribe"),
				tgbotapi.NewInlineKeyboardButtonData("Отписаться", "unsubscribe"),
			),
		)
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
	case "/help":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Раздел в разработке..."))
	case "/admin":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "https://www.youtube.com/watch?v=zXz0InOf-z0"))
	default:
		street, number := parseAddress(message.Text)
		if street == "" {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Введите корректный адрес"))
		}
		if action, exists := userAction[chatID]; exists {
			delete(userAction, chatID)
			if action == "subscribe" {
				lastPostID := getLastPostID()
				err := database.SaveAddress(chatID, street, number, lastPostID)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка при сохранении адреса"))
					return
				}
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "✅ Адрес успешно сохранен. Вы будете получать уведомления о новых постах."))
			}
			return
		}
		posts := getPosts()
		foundPosts := findPostsByData(posts, street, number)
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

func parseAddress(address string) (street, number string) {
	parts := strings.Fields(address)
	if len(parts) == 0 {
		return "", ""
	}

	lastPart := parts[len(parts)-1]
	if isNumber(lastPart) {
		number = lastPart
		if len(parts) > 1 {
			street = strings.Join(parts[:len(parts)-1], " ")
		}
	} else {
		street = strings.Join(parts, " ")
	}
	return
}

func isNumber(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return len(s) > 0
}

func getLastPostID() string {
	posts := getPosts()
	if len(posts) == 0 {
		return ""
	}
	return strconv.Itoa(int(posts[0].Id))
}

func extractTextFields(data map[string]interface{}) []Post {
	var posts []Post
	response, ok := data["response"].(map[string]interface{})
	if !ok {
		fmt.Println("Invalid response format: 'response' key not found or not a map")
		return posts
	}

	items, ok := response["items"].([]interface{})
	if !ok {
		fmt.Println("Invalid response format: 'items' key not found or not an array")
		return posts
	}

	for _, item := range items {
		post, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		text, ok := post["text"].(string)
		if ok && text != "" {
			date := time.Unix(int64(post["date"].(float64)), 0)
			posts = append(posts, Post{Id: post["id"].(float64), Text: text, Date: date})
		}
	}

	return posts
}

// вот тут бы разлучить логику получения постов и телеграм, отдельный пакет напрашивается
// телеграм - отвечает за телеграм (коммуникация), посты - за посты 
// много вызовов getPosts(), но достаточно 1 получения постов из АПИ, остальные можно просто хранить до следующего обновления
func getPosts() []Post { 
	// url хоста просится в константу
	url := fmt.Sprintf("https://api.vk.com/method/wall.get?access_token=%v&v=%v&domain=%s", config.LoadConfig().SrvAccessKey, config.LoadConfig().Version, config.LoadConfig().Domain)
	resp, err := http.Get(url)
	if err != nil {
		panic(fmt.Sprintf("Failed to make GET request: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to read response body: %v", err))
	}


	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		panic(fmt.Sprintf("Failed to unmarshal JSON: %v", err))
	}

	posts := extractTextFields(result)

	return posts
}

func findPostsByData(posts []Post, street, number string) []Post {
	var foundPosts []Post
	for _, post := range posts {
		matchesStreet := street == "" || strings.Contains(strings.ToLower(post.Text), strings.ToLower(street))
		matchesNumber := number == "" || strings.Contains(strings.ToLower(post.Text), strings.ToLower(number))
		if matchesStreet && matchesNumber {
			foundPosts = append(foundPosts, post)
		}
	}
	return foundPosts
}

// чекер и нотифаер - 2 разные сущности
func CheckAndNotifyUsers(bot *tgbotapi.BotAPI) {
	subscriptions, err := database.GetAllSubscriptions()
	if err != nil {
		log.Println("Ошибка при получении подписок:", err)
		return
	}

	posts := getPosts()

	for _, sub := range subscriptions {
		var newPosts []Post
		for _, post := range posts {
			if int64(post.Id) > sub.LastPostID {
				matchesStreet := sub.Street == "" || strings.Contains(strings.ToLower(post.Text), strings.ToLower(sub.Street))
				matchesNumber := sub.Number == "" || strings.Contains(strings.ToLower(post.Text), strings.ToLower(sub.Number))
				if matchesStreet && matchesNumber {
					newPosts = append(newPosts, post)
				}
			}
		}

		if len(newPosts) > 0 {
			for _, post := range newPosts {
				msg := tgbotapi.NewMessage(sub.ChatID, fmt.Sprintf("\n%s \nДата публикации: %v", post.Text, post.Date.Local().Format("02.01.2006 15:04")))
				bot.Send(msg)
			}

			lastPostID := newPosts[len(newPosts)-1].Id
			err := database.UpdateLastPostID(sub.ChatID, int64(lastPostID))
			if err != nil {
				log.Println("Ошибка при обновлении last_post_id:", err)
			}
		}
	}
}
