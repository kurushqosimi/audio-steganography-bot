package telegram

import (
	"errors"
	"fmt"
	"hide-in-audio-bot/clients/telegram"
	"hide-in-audio-bot/events"
	"hide-in-audio-bot/lib/e"
	"hide-in-audio-bot/storage"
	"log"
	"strings"
)

const (
	HideCmd    = "/hide"
	HelpCmd    = "/help"
	StartCmd   = "/start"
	ExtractCmd = "/extract"
)

func (p *Processor) doCmd(text string, chatID int, username string) error {
	text = strings.TrimSpace(text)

	log.Printf("got new command '%s' from '%s'", text, username)

	if isAudio(text) {
		return p.sendSaved(chatID)
	}

	switch text {
	case ExtractCmd:
		return p.DecodeInfo(chatID, username)
	case StartCmd:
		return p.sendHello(chatID)
	case HelpCmd:
		return p.sendHelp(chatID)
	case HideCmd:
		p.activeSession[username] = true
		return p.sendNextStep(chatID)
	default:
		active := p.activeSession[username]
		if active {
			return p.sendAudio(chatID, text, username)
		}
		return p.tg.SendMessage(chatID, msgUnknownCommand)
	}
}

func (p *Processor) DecodeInfo(chatID int, username string) error {
	_, audioData, err := p.storage.ReadWAV(username)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrDoesNotExists):
			return p.tg.SendMessage(chatID, msgDoesNotExists)
		default:
			return err
		}
	}
	extractedMessage := extractData(audioData)
	return p.tg.SendMessage(chatID, fmt.Sprintf("%s:%s", msgExtractedText, extractedMessage))
}

func (p *Processor) sendAudio(chatID int, textToHide string, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: send audio", err) }()

	err = p.prepareFile(chatID, textToHide, username)
	if err != nil {
		return err
	}

	if err = p.tg.SendAudio(chatID, msgSaved); err != nil {
		return err
	}

	return nil
}

func (p *Processor) sendSaved(chatID int) error {
	return p.tg.SendMessage(chatID, msgSaved)
}

func (p *Processor) sendNextStep(chatID int) error {
	return p.tg.SendMessage(chatID, msgSendTextForHiding)
}

func (p *Processor) sendHello(chatID int) error {
	return p.tg.SendMessage(chatID, msgHello)
}

func (p *Processor) sendHelp(chatID int) error {
	return p.tg.SendMessage(chatID, msgHelp)
}

func NewMessageSender(chatID int, tg *telegram.Client) func(string) error {
	return func(msg string) error {
		return tg.SendMessage(chatID, msg)
	}
}

func isAudio(text string) bool {
	return string(rune(events.Audio)) == text
}

func (p *Processor) prepareFile(chatID int, textToHide string, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't prepare file", err) }()

	fPath, err := p.storage.PrepareFile(username, textToHide)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrDoesNotExists):
			return p.tg.SendMessage(chatID, msgDoesNotExists)
		case errors.Is(err, storage.ErrTooLongMessage):
			return p.tg.SendMessage(chatID, msgTooLongText)
		default:
			return err
		}
	}

	err = p.tg.SendAudio(chatID, fPath)
	if err != nil {
		return err
	}

	return p.storage.Remove(username)
}

func extractData(audioData []byte) string {
	var messageBytes []byte

	for i := 0; i < len(audioData)/8; i++ {
		var byteVal byte
		for bit := 0; bit < 8; bit++ {
			byteVal |= (audioData[i*8+bit] & 1) << bit
		}

		if byteVal == 0 { // Если встретили конец строки
			break
		}
		messageBytes = append(messageBytes, byteVal)
	}

	return string(messageBytes)
}
