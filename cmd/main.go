package main

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/google/go-github/v39/github"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/oauth2"
	"os"
)

var (
	version    = "dev"     //nolint
	commit     = "none"    //nolint
	date       = "unknown" //nolint
	builtBy    = "unknown" //nolint
	log        = logrus.New()
	configPath string
)

// Client holds all the information when running. Has the config, context and expands the github.Client
type Client struct {
	*github.Client
	// Conf is the run time configuration
	Conf *Config
	// Context is a context.Context that is passed to the github.Client.
	Context context.Context
}

// Config holds the configuration that is parsed from the configuration file provided
type Config struct {
	// IntervalTime is the time in seconds between sending webhooks. Helps with rate limiting
	IntervalTime int64 `json:"interval_time" yaml:"interval_time" default:"3"`
	// Notifications are the NotificationReasons that are selected to be sent
	Notifications []string `json:"notification_types" yaml:"notification_types" default:"[assign,author,comment,invitation,manual,mention,review_requested,security_alert,state_change,subscribed,team_mention]"`
	// SleepDuration is the time in seconds between each pull of notifications. The sleep time includes the pull of notifications and any interval time
	SleepDuration int64 `json:"sleep_duration" yaml:"sleep_duration" default:"600"`
	// AllowUnread pulls all notifications and not just unread ones
	AllowUnread bool `json:"allow_unread" yaml:"allow_unread" default:"false"`
	//GithubToken is the Personal Access Token that is used to authenticate with GitHub. It is **HIGHLY** recommended to pass this as a environment variable
	GithubToken string `json:"github-token" yaml:"github-token" default:""`
	// WebhookURL is the URL to send all the notifications to
	WebhookURL string `json:"discord_url" yaml:"discord_url"`
}

// WebhookMessage is the layout of the message that gets POSTed
type WebhookMessage struct {
	//Username is the username that is webhook is sent with
	Username string `json:"username"`
	//Avatar is the URL of the image that the webhook is sent with
	Avatar string `json:"avatar_url,omitempty"`
	//Embeds are a list of Embeds that are sent in the webhook
	Embed []discordgo.MessageEmbed `json:"embeds,omitempty"`
}

func setup(conf Config) (c *Client) {
	ctx := context.TODO()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: conf.GithubToken})
	tc := oauth2.NewClient(ctx, ts)
	c = &Client{
		Client:  github.NewClient(tc),
		Conf:    &conf,
		Context: ctx,
	}
	_, _, err := c.Repositories.List(c.Context, "", nil)
	if err != nil {
		log.WithError(err).Fatalf("Got an error when checking authorization")
	}
	return
}

func main() {
	app := &cli.App{
		Name:    "Notification Relay",
		Usage:   fmt.Sprintf("Get notifications and relay them as webhooks POST\n, Commit: %s Date: %s Build By: %s", commit, date, builtBy),
		Version: version,
		Authors: []*cli.Author{
			{
				Name:  "Cyb3r-Jak3",
				Email: "cyb3rjak3@pm.me",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Value:       "./config.yml",
				Usage:       "Path to the configuration file",
				Destination: &configPath,
			},
			&cli.BoolFlag{
				Name:    "debug",
				EnvVars: []string{"LOG_LEVEL_DEBUG"},
				Aliases: []string{"d"},
			},
			&cli.BoolFlag{
				Name:    "verbose",
				EnvVars: []string{"LOG_LEVEL_INFO"},
				//Aliases: []string{"v"},
			},
			&cli.BoolFlag{
				Name:    "trace",
				EnvVars: []string{"LOG_LEVEL_TRACE"},
				Aliases: []string{"t"},
			},
		},
		Action: run,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.WithError(err).Fatal("Error running app")
		return
	}
}
