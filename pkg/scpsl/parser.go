package scpsl

import (
	"DiscordMBM/pkg/core"
	"DiscordMBM/pkg/discord"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andersfylling/disgord"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Monitor struct {
	Config *core.Config
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
	if config.SCPSLConfig.APIKey == nil || config.SCPSLConfig.AccountID == nil {
		return nil, errors.New("SCP:SL APIKey and AccountID are required")
	}

	_, err := os.ReadDir("cache")
	if err != nil {
		err = os.Mkdir("cache", os.ModePerm)
		if err != nil {
			return nil, errors.New("can not create cache folder")
		}
	}

	m := Monitor{Config: config}

	go func() {
		for {
			err := m.parseServers()
			if err != nil {
				log.Println(err)
			}

			time.Sleep(30 * time.Second)
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

		for {
			srvInfo, err := m.readServerInfo(serverID)
			if err != nil {
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

			time.Sleep(30 * time.Second)
		}
	})

	return
}

func (m *Monitor) readServerInfo(serverID int) (*ServerInfo, error) {
	data, err := os.ReadFile("cache/scpsl.json")
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error while reading scp:sl cache. Details: %s", err.Error()))
	}

	if len(data) == 0 {
		return nil, nil
	}

	var response APIResponse

	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error while parsing scp:sl cache. Details: %s", err.Error()))
	}

	for _, v := range response.Servers {
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

	file, err := os.Create("cache/scpsl.json")
	if err != nil {
		return errors.New(fmt.Sprintf("failed to create scp:sl cache data. Details: %s", err.Error()))
	}

	var response APIResponse

	err = json.Unmarshal(reqData, &response)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to parse scp:sl api response. Details: %s", err.Error()))
	}

	if !*response.Success {
		return errors.New(fmt.Sprintf("scp:sl api response status error: %s", *response.Error))
	}

	_, err = file.Write(reqData)
	err = file.Close()
	if err != nil {
		return errors.New(fmt.Sprintf("failed to write scp:sl cache data. Details: %s", err.Error()))
	}

	if m.Config.Logger {
		log.Println("Successfully got data from scp:sl api")
	}

	return nil
}
