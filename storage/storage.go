package storage

import "errors"

type Storage interface {
	Remove(username string) error
	Exists(username string) (bool, error)
	DownloadAudio(fileURL, username string) (err error)
	PrepareFile(username string, textToHide string) (fPath string, err error)
	ReadWAV(username string) ([]byte, []byte, error)
}

var (
	ErrDoesNotExists  = errors.New("user have not sent audio yet")
	ErrTooLongMessage = errors.New("too long message to hide")
)
