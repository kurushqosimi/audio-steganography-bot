package files

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"hide-in-audio-bot/lib/e"
	"hide-in-audio-bot/storage"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type Storage struct {
	basePath string
}

const (
	defaultPerm = 0774
	HeaderSize  = 44
)

func New(basePath string) Storage {
	return Storage{basePath: basePath}
}

func (s Storage) DownloadAudio(audioURL, username string) (err error) {
	if err := s.downloadFile(audioURL, username); err != nil {
		return e.Wrap("can't download audio", err)
	}

	return nil
}

func (s Storage) downloadFile(fileURL, username string) error {
	resp, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fPath := filepath.Join(s.basePath, username)

	if err := os.MkdirAll(fPath, defaultPerm); err != nil {
		return err
	}

	fName, err := fileName(username)
	if err != nil {
		return err
	}

	fPath = filepath.Join(fPath, fName)

	out, err := os.Create(fPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (s Storage) Remove(username string) error {
	fileName, err := fileName(username)
	if err != nil {
		return e.Wrap("can't remove file", err)
	}

	path := filepath.Join(s.basePath, username, fileName)

	if err := os.Remove(path); err != nil {
		msg := fmt.Sprintf("can't remove file %s", path)

		return e.Wrap(msg, err)
	}

	return nil
}

func (s Storage) Exists(username string) (bool, error) {
	fileName, err := fileName(username)
	if err != nil {
		return false, e.Wrap("can't check if file exits", err)
	}

	path := filepath.Join(s.basePath, username, fileName)

	switch _, err := os.Stat(path); {
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	case err != nil:
		msg := fmt.Sprintf("can't check if file %s exists", path)

		return false, e.Wrap(msg, err)
	}
	return true, nil
}

func (s Storage) PrepareFile(username string, textToHide string) (fPath string, err error) {
	isExits, err := s.Exists(username)
	if err != nil {
		return "", e.Wrap("can't search for an audio", err)
	}

	if !isExits {
		return "", storage.ErrDoesNotExists
	}

	fName, err := fileName(username)
	if err != nil {
		return "", e.Wrap("can't retrieve filename", err)
	}
	fPath = filepath.Join(s.basePath, username)
	fPath = filepath.Join(fPath, fName)
	header, audioData, err := s.readWAV(fPath)
	if err != nil {
		return "", e.Wrap("can't read wav file", err)
	}

	data, err := embedData(audioData, textToHide)
	if err != nil {
		return "", e.Wrap("can't embed data to file", err)
	}
	err = s.saveWAV(fPath, header, data)
	if err != nil {
		return "", e.Wrap("can't save generated audio", err)
	}
	return
}

func (s Storage) ReadWAV(username string) ([]byte, []byte, error) {
	isExits, err := s.Exists(username)
	if err != nil {
		return nil, nil, e.Wrap("can't search for an audio", err)
	}

	if !isExits {
		return nil, nil, storage.ErrDoesNotExists
	}

	fName, err := fileName(username)
	if err != nil {
		return nil, nil, e.Wrap("can't retrieve filename", err)
	}
	fPath := filepath.Join(s.basePath, username, fName)
	return s.readWAV(fPath)
}

func (s Storage) readWAV(filename string) ([]byte, []byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	// Читаем заголовок
	header := make([]byte, HeaderSize)
	if _, err := file.Read(header); err != nil {
		return nil, nil, err
	}

	// Читаем данные
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	return header, data[HeaderSize:], nil
}

// Функция для встраивания данных
func embedData(audioData []byte, message string) ([]byte, error) {
	message += "\x00"
	messageBytes := []byte(message)

	msgLen := len(messageBytes)
	audioLen := len(audioData)

	if msgLen*8 > audioLen {
		return nil, storage.ErrTooLongMessage
	}

	// Встраиваем каждый бит сообщения
	for i, byteVal := range messageBytes {
		for bit := 0; bit < 8; bit++ {
			audioData[i*8+bit] = (audioData[i*8+bit] & 0xFE) | ((byteVal >> bit) & 1)
		}
	}

	return audioData, nil
}

func (s Storage) saveWAV(filename string, header, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Сохраняем заголовок и измененные данные
	if _, err := file.Write(header); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}

	return nil
}

func fileName(p string) (string, error) {
	return hash(p)
}

func hash(p string) (string, error) {
	h := sha1.New()

	if _, err := io.WriteString(h, p); err != nil {
		return "", e.Wrap("can't calculate hash", err)
	}

	return fmt.Sprintf("%x.wav", h.Sum(nil)), nil
}
