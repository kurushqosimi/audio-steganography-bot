package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hide-in-audio-bot/lib/e"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

type Client struct {
	host     string
	basePath string
	client   http.Client
}

const (
	getUpdatesMethod   = "getUpdates"
	sendMessagesMethod = "sendMessage"
	getAudioMethod     = "getFile"
)

func New(host string, token string) *Client {
	return &Client{
		host:     host,
		basePath: newBasePath(token),
		client:   http.Client{},
	}
}

func newBasePath(token string) string {
	return "bot" + token
}

func (c *Client) Updates(offset int, limit int) (updates []Update, err error) {
	defer func() { err = e.WrapIfErr("can't get updates", err) }()

	q := url.Values{}
	q.Add("offset", strconv.Itoa(offset))
	q.Add("limit", strconv.Itoa(limit))

	data, err := c.doRequest(getUpdatesMethod, q)
	if err != nil {
		return nil, err
	}

	var res UpdatesResponse

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	fmt.Println("Data", string(data))
	return res.Result, nil
}

func (c *Client) GetAudio(fileID string) (urlStr string, err error) {
	defer func() { err = e.WrapIfErr("can't get updates", err) }()

	q := url.Values{}
	q.Add("file_id", fileID)

	data, err := c.doRequest(getAudioMethod, q)
	if err != nil {
		return "", e.Wrap("can't get audio file", err)
	}

	var res GetFileResponse

	if err := json.Unmarshal(data, &res); err != nil {
		return "", e.Wrap("can't parse response for getFile", err)
	}

	if !res.Ok {
		return "", e.Wrap("getFile failed", nil)
	}

	fileURL := url.URL{
		Scheme: "https",
		Host:   c.host,
		Path:   path.Join("file", c.basePath, res.Result.FilePath),
	}

	return fileURL.String(), nil
}

func (c *Client) SendMessage(chatID int, text string) error {
	q := url.Values{}
	q.Add("chat_id", strconv.Itoa(chatID))
	q.Add("text", text)

	_, err := c.doRequest(sendMessagesMethod, q)
	if err != nil {
		return e.Wrap("can't send message", err)
	}

	return nil
}

func (c *Client) SendAudio(chatID int, filePath string) (err error) {
	defer func() { err = e.WrapIfErr("can't send audio", err) }()
	u := url.URL{
		Scheme: "https",
		Host:   c.host,
		Path:   path.Join(c.basePath, "sendAudio"),
	}

	// Открываем файл для чтения
	file, err := os.Open(filePath)
	if err != nil {
		return e.Wrap("can't open audio file", err)
	}
	defer file.Close()

	// Формируем multipart запрос
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Добавляем поля
	if err := writer.WriteField("chat_id", strconv.Itoa(chatID)); err != nil {
		return e.Wrap("can't write chat_id field", err)
	}

	part, err := writer.CreateFormFile("audio", filepath.Base(filePath))
	if err != nil {
		return e.Wrap("can't create form file for audio", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return e.Wrap("can't copy file content to form", err)
	}

	if err := writer.Close(); err != nil {
		return e.Wrap("can't close writer", err)
	}

	// Создаем HTTP-запрос
	req, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return e.Wrap("can't create request for sendAudio", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Отправляем запрос
	resp, err := c.client.Do(req)
	if err != nil {
		return e.Wrap("can't send audio to Telegram", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return e.Wrap("can't read Telegram response", err)
	}

	// Проверяем успешность отправки
	var apiResponse struct {
		Ok bool `json:"ok"`
	}
	if err := json.Unmarshal(respBody, &apiResponse); err != nil {
		return e.Wrap("can't parse Telegram response", err)
	}

	if !apiResponse.Ok {
		return e.Wrap("Telegram API returned error", nil)
	}

	return nil
}

func (c *Client) doRequest(method string, query url.Values) (data []byte, err error) {
	defer func() { err = e.WrapIfErr("can't do request", err) }()

	u := url.URL{
		Scheme: "https",
		Host:   c.host,
		Path:   path.Join(c.basePath, method),
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = query.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
