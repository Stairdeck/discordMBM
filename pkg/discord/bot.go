package discord

import (
	"DiscordMBM/pkg/core"
	"errors"
	"fmt"
	"github.com/andersfylling/disgord"
)

type Bot struct {
	Client *disgord.Client
}

func InitBot(srvConfig core.ServerConfig) (*Bot, error) {
	if srvConfig.BotID == "" || srvConfig.BotToken == "" {
		return nil, errors.New(fmt.Sprintf("discord bot id or token can not be empty, server: %s", srvConfig.Name))
	}

	bot := &Bot{
		Client: disgord.New(disgord.Config{
			BotToken:     srvConfig.BotToken,
			ProjectName:  srvConfig.Name,
			DisableCache: true,
		}),
	}

	return bot, nil
}

func GetServerStatusPayload(isOnline bool, players string, maxPlayers string, gameMap *string) *disgord.UpdateStatusPayload {
	var payload disgord.UpdateStatusPayload

	var activities [1]disgord.Activity

	if !isOnline {
		activities[0] = disgord.Activity{
			Name: "offline",
			Type: 3,
		}

		payload = disgord.UpdateStatusPayload{
			AFK:    true,
			Game:   activities,
			Status: disgord.StatusDnd,
		}
	} else {
		var botStatus string
		var afk bool

		if players == "0" {
			botStatus = disgord.StatusIdle
			afk = true
		} else {
			afk = false
			botStatus = disgord.StatusOnline
		}

		var name string
		if gameMap == nil {
			name = fmt.Sprintf("%s/%s", players, maxPlayers)
		} else {
			name = fmt.Sprintf("%s/%s on %s", players, maxPlayers, *gameMap)
		}

		activities[0] = disgord.Activity{
			Name: name,
			Type: 0,
		}

		payload = disgord.UpdateStatusPayload{
			AFK:    afk,
			Game:   activities,
			Status: botStatus,
		}
	}

	return &payload
}
