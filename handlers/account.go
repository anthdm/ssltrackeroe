package handlers

import (
	"fmt"
	"os"

	"github.com/anthdm/ssltracker/data"
	"github.com/anthdm/ssltracker/settings"
	"github.com/anthdm/ssltracker/util"
	"github.com/gofiber/fiber/v2"
	"github.com/sujit-baniya/flash"
)

const maxNotifyUpfront = 356 / 2

type UpdateAccountParams struct {
	NotifyUpfront      int
	NotifyDefaultEmail string
	NotifyWebhookURL   string
}

func (p UpdateAccountParams) validate() fiber.Map {
	errors := fiber.Map{}
	if !util.IsValidEmail(p.NotifyDefaultEmail) {
		errors["notifyDefaultEmailError"] = "Please provide a valid email address"
	}
	if p.NotifyUpfront == 0 || p.NotifyUpfront > maxNotifyUpfront {
		errors["notifyUpfrontError"] = fmt.Sprintf("The amount of days to get notified can not be 0 and larger than %d days", maxNotifyUpfront)
	}
	if len(p.NotifyWebhookURL) != 0 {
		if !util.IsValidWebhook(p.NotifyWebhookURL) {
			errors["notifyWebhookURLError"] = fmt.Sprintf("%s is not a valid webhook URL", p.NotifyWebhookURL)
		}
	}
	return errors
}

func HandleAccountUpdate(c *fiber.Ctx) error {
	var params UpdateAccountParams
	if err := c.BodyParser(&params); err != nil {
		return err
	}
	if errors := params.validate(); len(errors) > 0 {
		return flash.WithData(c, errors).Redirect("/account")
	}
	user := getAuthenticatedUser(c)
	account, err := data.GetUserAccount(user.ID)
	if err != nil {
		return err
	}
	account.NotifyUpfront = params.NotifyUpfront
	account.NotifyDefaultEmail = params.NotifyDefaultEmail
	if err := data.UpdateAccount(account); err != nil {
		return err
	}
	return c.Redirect("/account")
}

func HandleAccountShow(c *fiber.Ctx) error {
	user := getAuthenticatedUser(c)
	account, err := data.GetUserAccount(user.ID)
	if err != nil {
		return err
	}
	context := fiber.Map{
		"account":           account,
		"customerPortalURL": os.Getenv("STRIPE_PORTAL_URL"),
		"settings":          settings.Account[account.Plan],
	}
	return c.Render("account/show", context)
}
