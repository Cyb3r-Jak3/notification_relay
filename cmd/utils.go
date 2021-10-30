package main

import (
	"fmt"
	"github.com/Cyb3r-Jak3/common/v3"
	"github.com/mcuadros/go-defaults"
)

func parseConfig(fileName string) (*Config, error) {
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
