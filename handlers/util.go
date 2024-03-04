package handlers

import (
	"github.com/anthdm/ssltracker/data"
	"github.com/gofiber/fiber/v2"
)

func isUserSignedIn(c *fiber.Ctx) bool {
	user := getAuthenticatedUser(c)
	return user != nil
}

func getAuthenticatedUser(c *fiber.Ctx) *data.User {
	value := c.Locals(localsUserKey)
	if user, ok := value.(*data.User); ok {
		return user
	}
	return nil
}
