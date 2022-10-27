package ut3

import (
	"DiscordMBM/pkg/core"
	"DiscordMBM/pkg/discord"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/sandertv/gophertunnel/query"
	"log"
	"strings"
	"time"
)

type Monitor struct {
	Config *core.Config
}

type ServerInfo struct {
	Players *string `json:"Players"`
}

func CreateMonitor(config *core.Config) (*Monitor, error) {
	m := Monitor{Config: config}

	return &m, nil
}

func (m *Monitor) Run(serverConfig core.ServerConfig) {
	if serverConfig.RefreshDelay <= 0 {
		log.Println(fmt.Sprintf("server %s must have valid refreshDelay property", serverConfig.Name))
		return
	}

	if serverConfig.Info["ip"] == nil {
		log.Println(fmt.Sprintf("server %s must have ip properties in info section", serverConfig.Name))
		return
	}

	ip := fmt.Sprintf("%s", serverConfig.Info["ip"])

	bot, err := discord.InitBot(serverConfig)
	if err != nil {
		log.Println(fmt.Sprintf("failed to set up discord bot on server %s. Details: %s", serverConfig.Name, err.Error()))
		return
	}

	defer bot.Client.Gateway().StayConnectedUntilInterrupted()

	bot.Client.Gateway().Ready(func(s disgord.Session, h *disgord.Ready) {
		if m.Config.Logger {
			log.Println(fmt.Sprintf("Successfully connected discord bot on server %s", serverConfig.Name))
		}

		for {
			srvInfo, err := m.readServerInfo(ip)
			if err != nil && m.Config.Logger {
				log.Println(fmt.Sprintf("Error while parsing server info of %s. Details: %s", serverConfig.Name, err.Error()))
			}

			if srvInfo == nil || srvInfo.Players == nil {
				if m.Config.Logger {
					log.Println(fmt.Sprintf("Server %s not found, trying again in %d seconds", serverConfig.Name, serverConfig.RefreshDelay))
				}

				err := s.UpdateStatus(discord.GetServerStatusPayload(false, "0", "0", nil))
				if err != nil {
					log.Println(err)
				}
			} else {
				playersInfo := strings.Split(*srvInfo.Players, "/")

				if m.Config.Logger {
					log.Println(
						fmt.Sprintf("Server %s found and has %s/%s players",
							serverConfig.Name, playersInfo[0], playersInfo[1]))
				}

				payload := discord.GetServerStatusPayload(true, playersInfo[0], playersInfo[1], nil)

				err := s.UpdateStatus(payload)
				if err != nil {
					log.Println(err)
				}
			}

			time.Sleep(time.Duration(serverConfig.RefreshDelay) * time.Second)
		}
	})

	return
}

func (m *Monitor) readServerInfo(ip string) (*ServerInfo, error) {
	info, err := query.Do(ip)
	if err != nil {
		return nil, err
	}

	players := fmt.Sprintf("%s/%s", info["numplayers"], info["maxplayers"])

	return &ServerInfo{Players: &players}, nil
}
