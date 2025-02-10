package database

import (
	"database/sql"
	"log"
)

type Subscription struct {
	ChatID     int64
	Street     string
	Number     string
	LastPostID int64
}

var db *sql.DB

func InitDB() {
	var err error
	db, err = sql.Open("sqlite3", "./subscriptions.db")
	if err != nil { // если нет, можно попробовать пересоздать файлик и заново подключиться, если не, то тогда уж помирать
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS subscriptions (
		id INTEGER PRIMARY KEY,
		chat_id INTEGER,
		street TEXT,
		number TEXT,
		last_post_id INTEGER
	)`)
	if err != nil {
		log.Fatal(err)
	}
}

func SaveAddress(chatID int64, street, number, lastPostID string) error {
	_, err := db.Exec("INSERT INTO subscriptions (chat_id, street, number, last_post_id) VALUES (?, ?, ?, ?)", chatID, street, number, lastPostID)
	return err
}

func SetSubscriptionState(chatID int64, state bool) error {
	_, err := db.Exec("UPDATE subscriptions SET is_subscribing = ? WHERE chat_id = ?", state, chatID)
	return err
}

func GetSubscriptionState(chatID int64) (bool, error) {
	var state bool
	err := db.QueryRow("SELECT is_subscribing FROM subscriptions WHERE chat_id = ?", chatID).Scan(&state)
	return state, err
}

func UpdateLastPostID(chatID int64, lastPostID int64) error {
	_, err := db.Exec("UPDATE subscriptions SET last_post_id = ? WHERE chat_id = ?", lastPostID, chatID)
	return err
}

func DeleteSubscription(chatID int64) error {
	_, err := db.Exec("DELETE FROM subscriptions WHERE chat_id = ?", chatID)
	return err
}

func GetAllSubscriptions() ([]Subscription, error) {
	rows, err := db.Query("SELECT chat_id, street, number, last_post_id FROM subscriptions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ChatID, &sub.Street, &sub.Number, &sub.LastPostID); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}
