package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/logs"
	"log"
)

func HandleDiscord(stop <-chan bool) {
	// todo check config

	s, err := discordgo.New("Bot " + config.App.Platform.Discord.BotToken)
	if err != nil {
		logs.Err.Println("Invalid bot parameters: %v", err)
	}

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) { log.Println("Discord is up!") })
	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by the bot itself
		// This isn't required in this specific example, but it's a good practice.
		if m.Author.ID == s.State.User.ID {
			return
		}
		// If the message is "ping" reply with "Pong!"
		if m.Content == "ping" {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Pong!")
		}

		// If the message is "pong" reply with "Ping!"
		if m.Content == "pong" {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Ping!")
		}
	})

	s.Identify.Intents = discordgo.IntentsGuildMessages
	s.LogLevel = discordgo.LogInformational

	err = s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	<-stop
	_ = s.Close()
	log.Println("Discord is shutting down.")
}
