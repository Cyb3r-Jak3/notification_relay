package main

import (
	"github.com/Cyb3r-Jak3/common/v3"
	"github.com/google/go-github/v39/github"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"time"
)

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
	conf, err := parseConfig(configPath)
	log.Tracef("Current config. %+v", conf)
	if err != nil {
		log.WithError(err).Fatal("Error parsing config")
		return err
	}
	client := setup(*conf)
	loop(client)
	return nil
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
					messages = append(messages, generateMessage(c, n))
				}
			}
			log.Debugf("Have %d messages to send", len(messages))
			for i := range messages {
				log.Tracef("Sending %+v", messages[i])
				if _, err := common.DoJSONRequest("POST", c.Conf.WebhookURL, &messages[i], nil); err != nil {
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
