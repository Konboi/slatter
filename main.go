package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
	"gopkg.in/yaml.v2"
)

var (
	configFile = flag.String("c", "config.yml", "config file")
)

type Config struct {
	Token          string   `yaml:"token"`
	GatherChannels []string `yaml:"gather_channels"`
	DailverChanel  string   `yaml:"daliver_channel"`
}

type Client struct {
	config          Config
	slackClient     *slack.Client
	ReplyCh         chan *slack.MessageEvent
	Channels        map[string]string
	Users           map[string]User
	Team            *slack.TeamInfo
	DaliverChnnelID string
}

type User struct {
	ID      string
	Name    string
	IconURL string
}

func main() {
	flag.Parse()

	config, err := NewConfig(*configFile)
	if err != nil {
		log.Println(err)
		return
	}

	cli, err := NewClient(config)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	go cli.Run()

	for {
		select {
		case msg := <-cli.ReplyCh:
			for _, ch := range cli.config.GatherChannels {
				if strings.Compare(cli.Channels[msg.Channel], ch) == 0 {
					go cli.DaliverMessage(msg)
				}
			}
		}
	}
}

func NewClient(config Config) (cli *Client, err error) {
	scli := slack.New(config.Token)

	team, err := scli.GetTeamInfo()

	channels, err := scli.GetChannels(true)
	if err != nil {
		return nil, err
	}

	channelMap := make(map[string]string)
	var daliver_channel_id string

	for _, v := range channels {
		channelMap[v.ID] = v.Name
		if strings.Compare(v.Name, config.DailverChanel) == 0 {
			daliver_channel_id = v.ID
		}
	}

	userMap := make(map[string]User)
	users, err := scli.GetUsers()
	for _, u := range users {
		userMap[u.ID] = User{
			ID:      u.ID,
			Name:    u.Name,
			IconURL: u.Profile.Image24,
		}
	}

	return &Client{
		config:          config,
		slackClient:     scli,
		ReplyCh:         make(chan *slack.MessageEvent),
		Channels:        channelMap,
		Users:           userMap,
		Team:            team,
		DaliverChnnelID: daliver_channel_id,
	}, nil
}

func NewConfig(path string) (config Config, err error) {
	config = Config{}

	data, err := ioutil.ReadFile(path)

	if err != nil {
		return config, err
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}

	if config.Token == "" {
		return config, fmt.Errorf("please set slack token")
	}

	return config, nil
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
				fmt.Printf("Error %s \n", ev.Error())
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
		Text:       fmt.Sprintf("%s from <%s|#%s>", msg.Text, fmt.Sprintf("https://%s.slack.com/archives/%s", cli.Team.Domain, cli.Channels[msg.Channel]), cli.Channels[msg.Channel]),
		Footer:     cli.Users[msg.User].Name,
		FooterIcon: cli.Users[msg.User].IconURL,
		Ts:         int64(ts),
	}
	attachments = append(attachments, attachment)

	_, _, err = cli.slackClient.PostMessage(cli.DaliverChnnelID, "", slack.PostMessageParameters{
		Attachments: attachments,
	})
	if err != nil {
		log.Println(err.Error())
	}
}
