package scpsl

import (
	"DiscordMBM/pkg/core"
	"DiscordMBM/pkg/discord"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/teivah/broadcast"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Monitor struct {
	Config *core.Config
	Relay  *broadcast.Relay[APIResponse]
}

type ServerInfo struct {
	ID      *int    `json:"ID"`
	Players *string `json:"Players"`
}

type APIResponse struct {
	Success  *bool        `json:"Success"`
	Error    *string      `json:"Error"`
	Cooldown *int         `json:"Cooldown"`
	Servers  []ServerInfo `json:"Servers"`
}

func CreateMonitor(config *core.Config) (*Monitor, error) {
	if config.SCPSLConfig.APIKey == nil || config.SCPSLConfig.AccountID == nil || config.SCPSLConfig.RefreshDelay == nil {
		return nil, errors.New("SCP:SL APIKey, AccountID and RefreshDelay are required")
	}

	if *config.SCPSLConfig.RefreshDelay <= 0 {
		return nil, errors.New("invalid RefreshDelay value")
	}

	m := Monitor{Config: config}
	m.Relay = broadcast.NewRelay[APIResponse]()

	go func() {
		for {
			err := m.parseServers()
			if err != nil {
				log.Println(err)
			}

			time.Sleep(time.Duration(*config.SCPSLConfig.RefreshDelay) * time.Second)
		}
	}()

	return &m, nil
}

func (m *Monitor) Run(serverConfig core.ServerConfig) {
	if serverConfig.Info["serverID"] == nil {
		log.Println(fmt.Sprintf("server %s must have serverID property in info section", serverConfig.Name))
		return
	}

	serverID, ok := serverConfig.Info["serverID"].(int)
	if !ok {
		log.Println(fmt.Sprintf("failed to get serverID on server %s", serverConfig.Name))
		return
	}

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

		l := m.Relay.Listener(1)
		for n := range l.Ch() {
			srvInfo, err := m.readServerInfo(n, serverID)
			if err != nil && m.Config.Logger {
				log.Println(err)
			}

			if srvInfo == nil || srvInfo.Players == nil {
				if m.Config.Logger {
					log.Println(fmt.Sprintf("Server %s(%d) not found, trying again in 30 seconds", serverConfig.Name, serverID))
				}

				err := s.UpdateStatus(discord.GetServerStatusPayload(false, "0", "0", nil))
				if err != nil {
					log.Println(err)
				}
			} else {
				if m.Config.Logger {
					log.Println(fmt.Sprintf("Server %s found and has %s players", serverConfig.Name, *srvInfo.Players))
				}

				playersInfo := strings.Split(*srvInfo.Players, "/")
				err := s.UpdateStatus(discord.GetServerStatusPayload(true, playersInfo[0], playersInfo[1], nil))
				if err != nil {
					log.Println(err)
				}
			}
		}
	})

	return
}

func (m *Monitor) readServerInfo(info APIResponse, serverID int) (*ServerInfo, error) {
	for _, v := range info.Servers {
		if *v.ID == serverID {
			return &v, nil
		}
	}

	return nil, nil
}

func (m *Monitor) serverInfoRequest() ([]byte, error) {
	apiUrl := fmt.Sprintf("https://api.scpslgame.com/serverinfo.php?key=%s&id=%d&players=true",
		*m.Config.SCPSLConfig.APIKey, *m.Config.SCPSLConfig.AccountID)

	resp, err := http.Get(apiUrl)

	if err != nil {
		log.Println(fmt.Sprintf("scp:sl api request err: %s", err.Error()))
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(fmt.Sprintf("scp:sl api request err: %s", err.Error()))
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		log.Println(fmt.Sprintf("scp:sl api request err: %s", err.Error()))
		return nil, err
	}

	return data, nil
}

func (m *Monitor) parseServers() error {
	if m.Config.Logger {
		log.Println("Requesting scp:sl servers data")
	}

	reqData, err := m.serverInfoRequest()
	if err != nil {
		return errors.New(fmt.Sprintf("failed to request scp:sl api. Details: %s", err.Error()))
	}

	var response APIResponse

	err = json.Unmarshal(reqData, &response)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to parse scp:sl api response. Details: %s", err.Error()))
	}

	if !*response.Success {
		return errors.New(fmt.Sprintf("scp:sl api response status error: %s", *response.Error))
	}

	m.Relay.Notify(response)

	if m.Config.Logger {
		log.Println("Successfully got data from scp:sl api")
	}

	return nil
}
