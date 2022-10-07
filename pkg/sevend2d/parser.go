package sevend2d

import (
	"DiscordMBM/pkg/core"
	"DiscordMBM/pkg/discord"
	"errors"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/gorcon/telnet"
	"log"
	"regexp"
	"strconv"
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
	if serverConfig.Info["telnetIP"] == nil ||
		serverConfig.Info["telnetPassword"] == nil ||
		serverConfig.Info["maxPlayers"] == nil {
		log.Println(
			fmt.Sprintf(
				"server %s must have telnetIP, telnetPassword and maxPlayers properties in info section",
				serverConfig.Name))
		return
	}

	ip := fmt.Sprintf("%s", serverConfig.Info["telnetIP"])
	password := fmt.Sprintf("%s", serverConfig.Info["telnetPassword"])

	conn, err := telnet.Dial(ip, password)
	if err != nil {
		log.Println(fmt.Sprintf("failed to set up parser on server %s. Details: %s", serverConfig.Name, err.Error()))
		return
	}
	defer conn.Close()

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
			srvInfo, err := m.readServerInfo(conn)
			if err != nil {
				log.Println(err)
			}

			if srvInfo == nil || srvInfo.Players == nil {
				if m.Config.Logger {
					log.Println(fmt.Sprintf("Server %s not found, trying again in 30 seconds", serverConfig.Name))
				}

				err := s.UpdateStatus(discord.GetServerStatusPayload(false, "0", "0", nil))
				if err != nil {
					log.Println(err)
				}
			} else {
				*srvInfo.Players = fmt.Sprintf("%s/%d", *srvInfo.Players, serverConfig.Info["maxPlayers"])

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

			time.Sleep(30 * time.Second)
		}
	})

	return
}

func (m *Monitor) readServerInfo(conn *telnet.Conn) (*ServerInfo, error) {
	info, err := conn.Execute("listplayers")
	if err != nil {
		return nil, err
	}

	// info string is "{loginfo}\nTotal of 0 in the game"
	re, _ := regexp.Compile(`Total of [0-9]+ in the game`)
	info = string(re.Find([]byte(info)))

	var players string
	if info != "" {
		re, _ := regexp.Compile(`[0-9]+`)
		res := string(re.Find([]byte(info)))

		playersNum, err := strconv.ParseInt(res, 10, 16)
		if err != nil {
			return nil, err
		}

		players = fmt.Sprintf("%d", playersNum)
	} else {
		return nil, errors.New(fmt.Sprintf("Unexpected telnet response. Details: %s", info))
	}

	return &ServerInfo{Players: &players}, nil
}
