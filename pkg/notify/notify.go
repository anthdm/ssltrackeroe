package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/anthdm/ssltracker/data"
	"github.com/anthdm/ssltracker/util"
	"github.com/gofiber/fiber/v2"
)

type Notifier interface {
	NotifyStatus(context.Context, data.TrackingAndAccount) error
	NotifyExpires(context.Context, data.TrackingAndAccount) error
	Kind() string
}

type EmailNotifier struct {
	to []string
}

func NewEmailNotifier(to []string) *EmailNotifier {
	return &EmailNotifier{
		to: to,
	}
}

func (n EmailNotifier) Kind() string { return "mailer send email" }

func (n *EmailNotifier) NotifyStatus(ctx context.Context, tracking data.TrackingAndAccount) error {
	fmt.Println("SEND EMAIL TO =>", tracking.Account.NotifyDefaultEmail)
	return nil
}

func (n *EmailNotifier) NotifyExpires(ctx context.Context, tracking data.TrackingAndAccount) error {
	fmt.Println("SEND EMAIL TO =>", tracking.Account.NotifyDefaultEmail)
	return nil
}

type SlackNotifier struct {
	webhookURL string
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
	}
}

func (n *SlackNotifier) Kind() string { return "Slack" }

func (n *SlackNotifier) NotifyExpires(ctx context.Context, tracking data.TrackingAndAccount) error {
	msg := fmt.Sprintf("Domain %s will expire in %s days", tracking.DomainName, util.DaysLeft(tracking.Expires))
	return postSlackMessage(n.webhookURL, msg)
}

func (n *SlackNotifier) NotifyStatus(ctx context.Context, tracking data.TrackingAndAccount) error {
	msg := fmt.Sprintf("Domain %s has a non healthy status: %s", tracking.DomainName, tracking.Status)
	return postSlackMessage(n.webhookURL, msg)
}

func postSlackMessage(url string, msg string) error {
	body := fiber.Map{
		"text": msg,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, fiber.MIMEApplicationJSON, bytes.NewReader(b))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned non 200 response: %d", resp.StatusCode)
	}
	return nil
}
