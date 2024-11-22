package main

import (
	"flag"
	tgClient "hide-in-audio-bot/clients/telegram"
	event_consumer "hide-in-audio-bot/consumer/event-consumer"
	"hide-in-audio-bot/events/telegram"
	"hide-in-audio-bot/storage/files"
	"log"
)

const (
	tgBotHost   = "api.telegram.org"
	storagePath = "files_storage"
	batchSize   = 100
)

func main() {
	eventsProcessor := telegram.New(
		tgClient.New(
			tgBotHost,
			mustToken(),
		),
		files.New(storagePath),
	)

	log.Println("service started")

	consumer := event_consumer.New(eventsProcessor, eventsProcessor, batchSize)
	if err := consumer.Start(); err != nil {
		log.Fatal("service is stopped", err)
	}
	// consumer.Start(fetcher, processor)
}

func mustToken() string {
	token := flag.String(
		"tg-bot-token",
		"",
		"token to access telegram bot",
	)

	flag.Parse()

	if token == nil {
		log.Fatal("token is not specified")
	}

	return *token
}
