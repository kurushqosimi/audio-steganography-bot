package telegram

import (
	"errors"
	"hide-in-audio-bot/clients/telegram"
	"hide-in-audio-bot/events"
	"hide-in-audio-bot/lib/e"
	"hide-in-audio-bot/storage"
)

type Processor struct {
	tg            *telegram.Client
	offset        int
	storage       storage.Storage
	activeSession map[string]bool
}

type Meta struct {
	ChatID   int
	Username string
}

var (
	ErrUnknownEventType = errors.New("unknown event type")
	ErrUnknownMetaType  = errors.New("unknown meta type")
)

func New(client *telegram.Client, storage storage.Storage) *Processor {
	return &Processor{
		tg:            client,
		storage:       storage,
		activeSession: make(map[string]bool, 50),
	}
}

func (p *Processor) Fetch(limit int) ([]events.Event, error) {
	updates, err := p.tg.Updates(p.offset, limit)
	if err != nil {
		return nil, e.Wrap("can't get events", err)
	}

	if len(updates) == 0 {
		return nil, nil
	}

	res := make([]events.Event, 0, len(updates))

	for _, u := range updates {
		res = append(res, event(u))
	}

	p.offset = updates[len(updates)-1].ID + 1

	return res, nil
}

func (p *Processor) Process(event events.Event) error {
	switch event.Type {
	case events.Message:
		return p.processMessage(event)
	case events.Audio:
		return p.processAudio(event)
	default:
		return e.Wrap("can't process", ErrUnknownEventType)
	}
}

func (p *Processor) processMessage(event events.Event) error {
	meta, err := meta(event)
	if err != nil {
		return e.Wrap("can't process message", err)
	}

	if err := p.doCmd(event.Text, meta.ChatID, meta.Username); err != nil {
		return e.Wrap("can't process message", err)
	}

	return nil
}

func (p *Processor) processAudio(event events.Event) error {
	meta, err := meta(event)
	if err != nil {
		return e.Wrap("can't process audio", err)
	}

	audioURL, err := p.tg.GetAudio(event.AudioID)
	if err != nil {
		return e.Wrap("can't get audio", err)
	}

	if err := p.storage.DownloadAudio(audioURL, meta.Username); err != nil {
		return err
	}

	if err := p.doCmd(string(rune(event.Type)), meta.ChatID, meta.Username); err != nil {
		return e.Wrap("can't process audio", err)
	}
	return nil
}

func meta(event events.Event) (Meta, error) {
	res, ok := event.Meta.(Meta)
	if !ok {
		return Meta{}, e.Wrap("can't get meta", ErrUnknownMetaType)
	}

	return res, nil
}

func event(upd telegram.Update) events.Event {
	updType := fetchType(upd)

	res := events.Event{
		Type: updType,
		Text: fetchText(upd),
	}

	if updType == events.Message {
		res.Meta = Meta{
			ChatID:   upd.Message.Chat.ID,
			Username: upd.Message.From.Username,
		}
	}

	if updType == events.Audio {
		res.Meta = Meta{
			ChatID:   upd.Message.Chat.ID,
			Username: upd.Message.From.Username,
		}
		res.AudioID = fetchAudioID(upd)
	}

	return res
}

func fetchAudioID(upd telegram.Update) string {
	if upd.Message != nil && upd.Message.Document != nil {
		return upd.Message.Document.FileID
	}

	if upd.Message != nil && upd.Message.Audio != nil {
		return upd.Message.Audio.FileID
	}

	return ""
}

func fetchText(upd telegram.Update) string {
	if upd.Message == nil {
		return ""
	}

	return upd.Message.Text
}

func fetchAudio(upd telegram.Update) telegram.Document {
	if upd.Message != nil && upd.Message.Document != nil {
		return *upd.Message.Document
	}
	return telegram.Document{}
}

func fetchType(upd telegram.Update) events.Type {
	if upd.Message == nil {
		return events.Unknown
	}
	if upd.Message.Document != nil || upd.Message.Audio != nil {
		return events.Audio
	}
	return events.Message
}
