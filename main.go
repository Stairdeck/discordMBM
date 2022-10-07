package main

import (
	"DiscordMBM/pkg/core"
	"DiscordMBM/pkg/minecraft"
	"DiscordMBM/pkg/scpsl"
	"DiscordMBM/pkg/sevend2d"
	"DiscordMBM/pkg/source"
	"DiscordMBM/pkg/ut3"
	"fmt"
	"log"
	"os"
	"os/signal"
)

type Monitors struct {
	SCPSL    *scpsl.Monitor
	Source   *source.Monitor
	UT3      *ut3.Monitor
	MC       *minecraft.Monitor
	SevenD2D *sevend2d.Monitor
}

func main() {
	config, err := core.ParseConfig("config.yml")

	if err != nil {
		log.Fatalln(err)
	}

	var monitors Monitors

	for _, server := range config.Servers {
		switch server.Game {
		case "scpsl":
			if monitors.SCPSL == nil {
				monitors.SCPSL, err = scpsl.CreateMonitor(config)

				if err != nil {
					log.Fatalln(err)
				}
			}

			log.Println(fmt.Sprintf("Running %s server", server.Name))
			go monitors.SCPSL.Run(server)
		case "source":
			if monitors.Source == nil {
				monitors.Source, err = source.CreateMonitor(config)

				if err != nil {
					log.Fatalln(err)
				}
			}

			log.Println(fmt.Sprintf("Running %s server", server.Name))
			go monitors.Source.Run(server)
		case "mc":
			if monitors.MC == nil {
				monitors.MC, err = minecraft.CreateMonitor(config)

				if err != nil {
					log.Fatalln(err)
				}
			}

			log.Println(fmt.Sprintf("Running %s server", server.Name))
			go monitors.MC.Run(server)
		case "7d2d":
			if monitors.SevenD2D == nil {
				monitors.SevenD2D, err = sevend2d.CreateMonitor(config)

				if err != nil {
					log.Fatalln(err)
				}
			}

			log.Println(fmt.Sprintf("Running %s server", server.Name))
			go monitors.SevenD2D.Run(server)
		case "ut3":
			if monitors.UT3 == nil {
				monitors.UT3, err = ut3.CreateMonitor(config)

				if err != nil {
					log.Fatalln(err)
				}
			}

			log.Println(fmt.Sprintf("Running %s server", server.Name))
			go monitors.UT3.Run(server)
		}
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	<-sc
}
