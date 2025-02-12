package post

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
	"waterTelegram/config"
)

type Post struct {
	Id   float64   `json:"id"`
	Text string    `json:"text"`
	Date time.Time `json:"date"`
}

func GetPostsFromAPI() []Post {
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

	posts := ExtractTextFields(result)

	return posts
}

func FindPostsByData(posts []Post, street, number string) []Post {
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

func ExtractTextFields(data map[string]interface{}) []Post {
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

func ParseAddress(address string) (street, number string) {
	parts := strings.Fields(address)
	if len(parts) == 0 {
		return "", ""
	}

	if len(parts) == 1 {
		return parts[0], ""
	}

	lastElement := parts[len(parts)-1]
	hasDigit := strings.IndexFunc(lastElement, unicode.IsDigit) != -1
	if hasDigit {
		number = lastElement
		street = strings.Join(parts[:len(parts)-1], " ")
	} else {
		street = address
		number = ""
	}

	return street, number
}
