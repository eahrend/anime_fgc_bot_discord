package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/eahrend/anime_fgc_bot/common/models"
	"log"
	"net/http"
	"os"
	"os/signal"
)

var s *discordgo.Session

var err error

func init() {
	botToken := os.Getenv("BOT_TOKEN")
	s, err = discordgo.New(fmt.Sprintf("Bot %s", botToken))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

func main() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})
	err = s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	cmd := &discordgo.ApplicationCommand{
		Description: "Frame Data Reader",
		Name:        "fd",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "game",
				Description: "Name of the game",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "character",
				Description: "Name of the character",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "type",
				Description: "Type of move",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Special Moves",
						Value: "specials",
					},
					{
						Name:  "Super Moves",
						Value: "supers",
					},
					{
						Name:  "Throws",
						Value: "throws",
					},
					{
						Name:  "Normals",
						Value: "normals",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Name or input of the move",
				Required:    true,
			},
		},
	}
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		characterName := i.ApplicationCommandData().Options[1].StringValue()
		moveName := i.ApplicationCommandData().Options[3].StringValue()
		botUrl := fmt.Sprintf("https://fgc.ngrok.io/api/v1/character/%s/%s/%s/%s",
			i.ApplicationCommandData().Options[0].StringValue(),
			i.ApplicationCommandData().Options[1].StringValue(),
			i.ApplicationCommandData().Options[2].StringValue(),
			i.ApplicationCommandData().Options[3].StringValue(),
		)
		fmt.Println("Bot URL:", botUrl)
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodGet, botUrl, nil)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("failed to create request: %s", err.Error()),
				},
			})
			return
		}
		res, err := client.Do(req)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("failed to get data: %s", err.Error()),
				},
			})
			return
		}
		msgformat :=
			`Frame Data for %s %s:
				> Guard: %s
				> Startup: %s
				> Active: %s
				> Recovery: %s
			`
		msgData := ""
		// TODO: Create character interfaces with toMsg() data so we don't have to decode here
		switch moveType := i.ApplicationCommandData().Options[2].StringValue(); moveType {
		// TODO: Add chargable boolean to character models
		case "normals":
			striveNormalMove := models.StriveCharacterNormalMove{}
			err = json.NewDecoder(res.Body).Decode(&striveNormalMove)
			if err != nil {
				panic(err)
			}
			msgData = fmt.Sprintf(msgformat, characterName, moveName, striveNormalMove.Guard, striveNormalMove.Startup, striveNormalMove.Active, striveNormalMove.Recovery)
			if len(striveNormalMove.ChargeDamage) > 0 {
				msgData = fmt.Sprintf(`%s
					> Charged Startup: %s
					> Charged Damage: %s
					> Charged On Block: %s
				`, fmt.Sprintf(msgformat, characterName, moveName, striveNormalMove.Guard, striveNormalMove.Startup, striveNormalMove.Active, striveNormalMove.Recovery),
					striveNormalMove.ChargeStartup,
					striveNormalMove.ChargeDamage,
					striveNormalMove.ChargeOnBlock)
			}
		case "specials":
			striveSpecialMove := models.StriveCharacterSpecialMove{}
			err = json.NewDecoder(res.Body).Decode(&striveSpecialMove)
			if err != nil {
				panic(err)
			}
			msgData = fmt.Sprintf(msgformat, characterName, "Not yet implemented", moveName, striveSpecialMove.Startup, striveSpecialMove.Active, striveSpecialMove.Recovery)
		case "supers":
			striveSuperMove := models.StriveCharacterSuperMove{}
			err = json.NewDecoder(res.Body).Decode(&striveSuperMove)
			if err != nil {
				panic(err)
			}
			msgData = fmt.Sprintf(msgformat, characterName, "Not yet implemented", moveName, striveSuperMove.Startup, striveSuperMove.Active, striveSuperMove.Recovery)
		case "throws":
			striveThrowMove := models.StriveCharacterThrowMove{}
			err = json.NewDecoder(res.Body).Decode(&striveThrowMove)
			if err != nil {
				panic(err)
			}
			msgData = fmt.Sprintf(msgformat, characterName, "Not yet implemented", moveName, striveThrowMove.Startup, "", striveThrowMove.Recovery)
		default:
			panic(moveType)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msgData,
			},
		})
	})
	_, err = s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
	if err != nil {
		panic(err)
	}
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Gracefully shutdowning")
}
