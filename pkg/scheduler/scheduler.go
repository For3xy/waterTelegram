package scheduler

import (
	"log"
	"time"
	"waterTelegram/config"
	"waterTelegram/pkg/post"
	"waterTelegram/pkg/repository"
)

func StartScheduler() {
	ticker := time.NewTicker(time.Duration(config.LoadConfig().AutosaveInterval) * time.Second)
	go func() {
		updatePosts()
		for range ticker.C {
			updatePosts()
		}
	}()
}

func updatePosts() {
	apiPosts := post.GetPostsFromAPI()
	cachedPosts := repository.GetPosts()
	if cachedPosts == nil || len(cachedPosts) == 0 {
		repository.UpdateCache(apiPosts)
		log.Printf("[%s] Кэш был пуст. Записано %d постов.", time.Now().Format("2006-01-02 15:04"), len(apiPosts))
		return
	}

	if len(apiPosts) == 0 {
		repository.RefreshCacheTime()
		log.Printf("[%s] API не вернул постов. Обновлено время актуальности кэша.", time.Now().Format("2006-01-02 15:04:05"))
		return
	}

	cachedTopID := cachedPosts[0].Id
	apiTopID := apiPosts[0].Id

	if apiTopID > cachedTopID {
		var newPosts []post.Post
		for _, p := range apiPosts {
			if p.Id > cachedTopID {
				newPosts = append(newPosts, p)
			} else {
				break
			}
		}

		updatedCache := append(newPosts, cachedPosts...)
		repository.UpdateCache(updatedCache)
		log.Printf("[%s] Найдено %d новых постов. Обновлено общее число постов: %d.", time.Now().Format("2006-01-02 15:04"), len(newPosts), len(updatedCache))
	} else {
		repository.RefreshCacheTime()
		log.Printf("[%s] Новых постов не обнаружено.Обновлено время актуальности кэша.", time.Now().Format("2006-01-02 15:04"))
	}
}
