package models

import "time"

type (
	Session struct {
		Login     string
		SID       string
		ExpiresAt time.Time
	}

	UserItem struct {
		Login string `json:"login"`
	}

	AdvertItem struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Price       int64  `json:"price"`
		ImagePath   string `json:"image_path"`
	}

	Advert struct {
		ID           int64  `json:"id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		Price        int64  `json:"price"`
		ImagePath    string `json:"image_path"`
		CreatorLogin string `json:"creator_login"`
	}
)
