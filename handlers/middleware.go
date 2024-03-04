package handlers

import (
	"context"

	"github.com/anthdm/ssltracker/data"
	"github.com/anthdm/ssltracker/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/sujit-baniya/flash"
)

const localsUserKey = "user"

func WithFlash(c *fiber.Ctx) error {
	values := flash.Get(c)
	c.Locals("flash", values)
	return c.Next()
}

func WithViewHelpers(c *fiber.Ctx) error {
	c.Locals("activeFor", func(s string) (res string) {
		if c.Path() == s {
			return "active"
		}
		return ""
	})
	return c.Next()
}

func WithAuthenticatedUser(c *fiber.Ctx) error {
	c.Locals(localsUserKey, nil)
	client := createSupabaseClient()
	token := c.Cookies("accessToken")

	if len(token) == 0 {
		return c.Next()
	}

	user, err := client.Auth.User(context.Background(), token)
	if err != nil {
		logger.Log("error", "authentication error", "err", "probably invalid access token")
		c.ClearCookie("accessToken")
		return c.Redirect("/")
	}

	ourUser := &data.User{ID: user.ID, Email: user.Email}
	c.Locals(localsUserKey, ourUser)

	return c.Next()
}

func WithMustBeAuthenticated(c *fiber.Ctx) error {
	user := getAuthenticatedUser(c)
	if user == nil {
		return c.RedirectBack("/")
	}
	return c.Next()
}
