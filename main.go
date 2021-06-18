package main

import (
	"context"
	"fmt"
	"os"
	"time"

	common "github.com/Cyb3r-Jak3/common/go"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/google/go-github/v35/github"
	"github.com/mcuadros/go-defaults"
	discord "github.com/nickname32/discordhook"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/oauth2"
)

// NotificationReasons are valid GitHub notification reasons.
//
// https://docs.github.com/en/rest/reference/activity#notification-reasons
var NotificationReasons = []string{
	"assign",
	"author",
	"comment",
	"invitation",
	"manual",
	"mention",
	"review_requested",
	"security_alert",
	"state_change",
	"subscribed",
	"team_mention",
}

var log = logrus.New()
var configPath string

// Client holds all the information when running. Has the config, context and expands the github.Client
type Client struct {
	*github.Client
	// Conf is the run time configuration
	Conf    *Config
	// Context is a context.Context that is passed to the github.Client.
	Context context.Context
}

// Config holds the configuration that is parsed from the configuration file provided
type Config struct {
	// IntervalTime is the time in seconds between sending webhooks. Helps with rate limiting
	IntervalTime  int64    `json:"interval_time" yaml:"interval_time" default:"3"`
	// Notifications are the NotificationReasons that are selected to be sent
	Notifications []string `json:"notification_types" yaml:"notification_types" default:"[assign,author,comment,invitation,manual,mention,review_requested,security_alert,state_change,subscribed,team_mention]"`
	// SleepDuration is the time in seconds between each pull of notifications. The sleep time includes the pull of notifications and any interval time
	SleepDuration int64    `json:"sleep_duration" yaml:"sleep_duration" default:"600"`
	// AllowUnread pulls all notifications and not just unread ones
	AllowUnread   bool     `json:"allow_unread" yaml:"allow_unread" default:"false"`
	//GithubToken is the Personal Access Token that is used to authenticate with GitHub. It is **HIGHLY** recommended to pass this as a environment variable
	GithubToken   string `json:"github-token" yaml:"github-token" default:""`
	// WebhookURL is the URL to send all the notifications to
	WebhookURL    string `json:"discord_url" yaml:"discord_url"`
}
// WebhookMessage is the layout of the message that gets POSTed
type WebhookMessage struct {
	//Username is the username that is webhook is sent with
	Username string          `json:"username"`
	//Avatar is the URL of the image that the webhook is sent with
	Avatar   string          `json:"avatar_url,omitempty"`
	//Embeds are a list of Embeds that are sent in the webhook
	Embed    []discord.Embed `json:"embeds,omitempty"`
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
		log.WithError(err).Fatalf("Got an error when checking authication")
	}
	return
}

func generatemessage(c *Client, notification *github.Notification) WebhookMessage {
	req, _ := c.NewRequest("GET", *notification.Subject.LatestCommentURL, nil)
	comment := new(github.IssueComment)
	_, err := c.Do(c.Context, req, comment)
	if err != nil {
		log.WithError(err).Info("Error making request for Github comments")
		return WebhookMessage{}
	}
	converter := md.NewConverter("", true, nil)
	body, _ := converter.ConvertString(comment.GetBody())
	httpURL := comment.GetHTMLURL()
	if len(body) >= 1000 {
		body = body[:1000] + fmt.Sprintf("\n\n**[Click Here](%s)** to view message in web browser", httpURL)
	}
	return WebhookMessage{
		Avatar:   notification.Repository.Owner.GetAvatarURL(),
		Username: "Notification Relay",
		Embed: []discord.Embed{
			{
				Title:       fmt.Sprintf("New %s", notification.Subject.GetType()),
				URL:         httpURL,
				Description: body,
				Timestamp:   comment.UpdatedAt,
				Thumbnail: &discord.EmbedThumbnail{
					URL: comment.User.GetAvatarURL(),
				},
			},
		},
	}
}


func parseconfig(fileName string) (*Config, error) {
	conf := new(Config)
	defaults.SetDefaults(conf)
	if fileName != "" {
		err := common.ParseYamlOrJSON(fileName, conf)
		if err != nil {
			log.WithError(err)
			return nil, err
		}
		for _, v := range conf.Notifications {
			if !common.StringSearch(v, NotificationReasons) {
				return nil, fmt.Errorf("%s is not a valid notification reason", v)
			}
		}
	}
	if conf.WebhookURL == "" {
		conf.WebhookURL = common.GetEnvSecret("DISCORD_URL")
		if conf.WebhookURL == "" {
			return nil, fmt.Errorf("there was no discord webhook URL provided in either the config file or as an os environment variable")
		}
	}
	conf.GithubToken = common.GetEnvSecret("GITHUB_TOKEN")
	return conf, nil

}

func loop(c *Client) {
	var latestTime time.Time
	log.Debug("Starting loop")
	for {
		t0 := time.Now()
		notifications, _, err := c.Activity.ListNotifications(
			c.Context, &github.NotificationListOptions{All: false, Since: latestTime},
		)
		if err != nil {
			log.WithError(err).Warning("Error when listing notifications")
		}
		if len(notifications) == 0 {
			log.Debug("No unread notifications")
		} else {
			var messages []WebhookMessage
			for _, n := range notifications {
				log.Tracef("Notification Reason: %s. Configured reasons %v. Detected in slice %t", n.GetReason(), c.Conf.Notifications, common.StringSearch(n.GetReason(), c.Conf.Notifications))
				if !common.StringSearch(n.GetReason(), c.Conf.Notifications) {
					log.Debugf("Notification had reason %s and configured to ignore", n.GetReason())
				} else {
					messages = append(messages, generatemessage(c, n))
				}
			}
			log.Debugf("Have %d messages to send", len(messages))
			for _, n := range messages {
				log.Tracef("Sending %+v", n)
				if _, err := common.DoJSONRequest("POST", c.Conf.WebhookURL, &n, nil); err != nil {
					log.WithError(err).Error("Error sending discord payload")
				}
				time.Sleep(time.Duration(c.Conf.IntervalTime) * time.Second)
			}
		}
		t1 := time.Now()
		latestTime = t0
		sleepTime := time.Duration(c.Conf.SleepDuration)*time.Second - t1.Sub(t0)
		log.Debugf("Loop took %v to run. Sleeping for %v\n", t1.Sub(t0), sleepTime)
		// Sleep for the remainder of the time
		time.Sleep(sleepTime)
	}
}

func run(c *cli.Context) error {
	if c.Bool("trace") {
		log.SetLevel(logrus.TraceLevel)
	} else if c.Bool("debug") {
		log.SetLevel(logrus.DebugLevel)
	} else if c.Bool("verbose") {
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetLevel(logrus.WarnLevel)
	}
	log.Tracef("Trace %v, Debug %v, Verbose %v", c.Bool("trace"), c.Bool("debug"), c.Bool("verbose"))
	log.Debugf("Log Level set to %v", log.Level)
	conf, err := parseconfig(configPath)
	log.Tracef("Current config. %+v", conf)
	if err != nil {
		log.WithError(err).Fatal("Error parsing config")
		return err
	}
	client := setup(*conf)
	loop(client)
	return nil
}

func main() {
	app := &cli.App{
		Name:  "Notification Relay",
		Usage: "Get notifications and relay them as webhooks POST",
		Authors: []*cli.Author{
			{
				Name:  "Cyb3r-Jak3",
				Email: "jake@jwhite.network",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "file",
				Value:       "/config.yml",
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
				Aliases: []string{"v"},
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
