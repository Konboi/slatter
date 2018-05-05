package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Token          string   `yaml:"token"`
	GatherChannels []string `yaml:"gather_channels"`
	DailverChannel string   `yaml:"daliver_channel"`
}

type Client struct {
	config           Config
	slackClient      *slack.Client
	ReplyCh          chan *slack.MessageEvent
	Channels         map[string]slack.Channel
	Users            map[string]slack.User
	Team             *slack.TeamInfo
	DaliverChannelID string
}

type User struct {
	ID      string
	Name    string
	IconURL string
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "config.yml", "config file")
	flag.Parse()

	config, err := NewConfig(configFile)
	if err != nil {
		log.Fatalln("error new config", err.Error())
	}

	cli, err := NewClient(config)
	if err != nil {
		log.Fatalln("error new client", err.Error())
	}
	go cli.Run()

	for {
		select {
		case msg := <-cli.ReplyCh:
			for _, ch := range cli.config.GatherChannels {
				if strings.Compare(cli.Channels[msg.Channel].Name, ch) == 0 {
					go cli.DaliverMessage(msg)
				}
			}
		}
	}
}

func NewConfig(path string) (config Config, err error) {
	config = Config{}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return config, errors.Wrap(err, "error read file")
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, errors.Wrap(err, "error unmarshal")
	}

	if config.Token == "" {
		return config, fmt.Errorf("please set slack token")
	}

	return config, nil
}

func NewClient(config Config) (cli *Client, err error) {
	scli := slack.New(config.Token)

	team, err := scli.GetTeamInfo()
	if err != nil {
		return nil, errors.Wrap(err, "error get team info")
	}

	channels, err := scli.GetChannels(true)
	if err != nil {
		return nil, errors.Wrap(err, "error get channels")
	}

	channelMap := make(map[string]slack.Channel)
	var daliver_channel_id string

	for _, c := range channels {
		channelMap[c.ID] = c
		if strings.Compare(c.Name, config.DailverChannel) == 0 {
			daliver_channel_id = c.ID
		}
	}

	userMap := make(map[string]slack.User)
	users, err := scli.GetUsers()
	if err != nil {
		return nil, errors.Wrap(err, "error get users")
	}

	for _, u := range users {
		userMap[u.ID] = u
	}

	return &Client{
		config:           config,
		slackClient:      scli,
		ReplyCh:          make(chan *slack.MessageEvent),
		Channels:         channelMap,
		Users:            userMap,
		Team:             team,
		DaliverChannelID: daliver_channel_id,
	}, nil
}

func (cli *Client) Run() {
	fmt.Println("start application")

	rtm := cli.slackClient.NewRTM()

	go rtm.ManageConnection()

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
			case *slack.ConnectedEvent:
				fmt.Println("slack connected")
			case *slack.MessageEvent:
				cli.ReplyCh <- ev
			case *slack.RTMError:
				fmt.Printf("error rtm message %s \n", ev.Error())
			}
		}
	}
}

func (cli *Client) DaliverMessage(msg *slack.MessageEvent) {
	var attachments []slack.Attachment
	ts, err := strconv.ParseFloat(msg.Timestamp, 64)
	if err != nil {
		log.Println(err.Error())
	}

	attachment := slack.Attachment{
		Text:       fmt.Sprintf("%s from <%s|#%s>", msg.Text, fmt.Sprintf("https://%s.slack.com/archives/%s", cli.Team.Domain, cli.Channels[msg.Channel].Name), cli.Channels[msg.Channel].Name),
		Footer:     cli.Users[msg.User].Name,
		FooterIcon: cli.Users[msg.User].Profile.Image24,
		Ts:         json.Number(fmt.Sprintf("%d", ts)),
	}
	attachments = append(attachments, attachment)

	_, _, err = cli.slackClient.PostMessage(cli.DaliverChannelID, "", slack.PostMessageParameters{
		Attachments: attachments,
	})
	if err != nil {
		log.Println(err.Error())
	}
}
