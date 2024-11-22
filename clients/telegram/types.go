package telegram

import "time"

type UpdatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type Update struct {
	ID      int              `json:"update_id"`
	Message *IncomingMessage `json:"message"`
}

type GetFileResponse struct {
	Ok     bool `json:"ok"`
	Result File `json:"result"`
}

type File struct {
	FilePath string `json:"file_path"`
}

type IncomingMessage struct {
	Text     string    `json:"text"`
	From     From      `json:"from"`
	Chat     Chat      `json:"chat"`
	Document *Document `json:"document"`
	Audio    *Audio    `json:"audio"`
}

type Audio struct {
	Duration time.Duration `json:"duration,omitempty"`
	FileName string        `json:"file_name,omitempty"`
	FileID   string        `json:"file_id,omitempty"`
	MimeType string        `json:"mime_type,omitempty"`
	FileSize int           `json:"file_size,omitempty"`
}

type Document struct {
	FileID    string `json:"file_id"`
	Performer string `json:"performer,omitempty"`
	Title     string `json:"title,omitempty"`
	FileName  string `json:"file_name,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	FileSize  int    `json:"file_size,omitempty"`
}

type From struct {
	Username string `json:"username"`
}

type Chat struct {
	ID int `json:"id"`
}
