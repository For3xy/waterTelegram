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
	if err != nil {
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

func UpdateLastPostID(chatID int64, lastPostID int64) error {
	_, err := db.Exec("UPDATE subscriptions SET last_post_id = ? WHERE chat_id = ?", lastPostID, chatID)
	return err
}

func DeleteSubscription(chatID int64) error {
	_, err := db.Exec("DELETE FROM subscriptions WHERE chat_id = ?", chatID)
	return err
}

func DeleteManySubscription(chatID int64, street, number string) error {
	var query string
	var args []interface{}
	if number != "" {
		query = "DELETE FROM subscriptions WHERE chat_id = ? AND street = ? AND number = ?"
		args = []interface{}{chatID, street, number}
	} else {
		query = "DELETE FROM subscriptions WHERE chat_id = ? AND street = ? "
		args = []interface{}{chatID, street}
	}
	_, err := db.Exec(query, args...)
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

func GetAllSubscriptionsByChatID(chatID int64) ([]Subscription, error) {
	rows, err := db.Query("SELECT street, number, last_post_id FROM subscriptions WHERE chat_id = ?", chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.Street, &sub.Number, &sub.LastPostID); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, sub)
	}
	return subscriptions, nil
}
