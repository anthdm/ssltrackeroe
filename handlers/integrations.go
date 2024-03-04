package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/anthdm/ssltracker/data"
	"github.com/gofiber/fiber/v2"
	"github.com/slack-go/slack"
)

func HandleIntegrations(c *fiber.Ctx) error {
	user := getAuthenticatedUser(c)
	account, err := data.GetUserAccount(user.ID)
	if err != nil {
		return err
	}
	isSlackConnected := len(account.SlackAccessToken) > 0
	slackURL := fmt.Sprintf("https://slack.com/oauth/v2/authorize?scope=incoming-webhook&client_id=%s", os.Getenv("SLACK_CLIENT_ID"))
	context := fiber.Map{
		"slackConnectURL":  slackURL,
		"isSlackConnected": isSlackConnected,
	}
	return c.Render("integrations/index", context)
}

func HandleSlackCallback(c *fiber.Ctx) error {
	var (
		code        = c.Query("code")
		secret      = os.Getenv("SLACK_SECRET")
		clientID    = os.Getenv("SLACK_CLIENT_ID")
		redirectURL = os.Getenv("SLACK_REDIRECT_URL")
	)

	resp, err := slack.GetOAuthV2Response(http.DefaultClient, clientID, secret, code, redirectURL)
	if err != nil {
		return err
	}

	user := getAuthenticatedUser(c)
	account, err := data.GetUserAccount(user.ID)
	if err != nil {
		return err
	}

	account.SlackAccessToken = resp.AccessToken
	account.SlackChannelID = resp.IncomingWebhook.ChannelID
	account.SlackWebhookURL = resp.IncomingWebhook.URL
	if err := data.UpdateAccount(account); err != nil {
		return err
	}

	return c.Redirect("/integrations")
}
