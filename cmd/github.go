package main

import (
	"fmt"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/google/go-github/v39/github"
	"github.com/bwmarrin/discordgo"
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

func generateMessage(c *Client, notification *github.Notification) WebhookMessage {
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
		Embed: []discordgo.MessageEmbed{
			{
				Title:       fmt.Sprintf("New %s", notification.Subject.GetType()),
				URL:         httpURL,
				Description: body,
				Timestamp:   comment.UpdatedAt.Format("RFC1123"),
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: comment.User.GetAvatarURL(),
				},
			},
		},
	}
}
