// Package lang contains generated Go types from OpenAPI schema.
package lang

import "time"

type Types struct {
	BlogComment *BlogComment `json:"BlogComment,omitempty"`
	BlogPost    *BlogPost    `json:"BlogPost,omitempty"`
	BlogUser    *BlogUser    `json:"BlogUser,omitempty"`
}

type BlogComment struct {
	AuthorID  string    `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BlogPost struct {
	AuthorID  string    `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Featured  bool      `json:"featured"`
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BlogUser struct {
	Balance     float64                `json:"balance"`
	Bio         *string                `json:"bio,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	DisplayName string                 `json:"display_name"`
	Email       string                 `json:"email"`
	ID          string                 `json:"id"`
	Level       int64                  `json:"level"`
	Money       string                 `json:"money"`
	TheDate     string                 `json:"the_date"`
	TheDay      time.Time              `json:"the_day"`
	TheFile     string                 `json:"the_file"`
	TheObject   map[string]interface{} `json:"the_object"`
	TheTime     string                 `json:"the_time"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Username    string                 `json:"username"`
}
