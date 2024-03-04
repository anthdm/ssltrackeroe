package handlers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthdm/ssltracker/data"
	"github.com/anthdm/ssltracker/logger"
	"github.com/anthdm/ssltracker/util"
	"github.com/gofiber/fiber/v2"
	"github.com/nedpals/supabase-go"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/checkout/session"
	"github.com/sujit-baniya/flash"
)

func HandleGetSignup(c *fiber.Ctx) error {
	return c.Render("auth/signup", fiber.Map{})
}

type SignupParams struct {
	Email    string
	Fullname string
	Password string
}

func (p SignupParams) Validate() fiber.Map {
	data := fiber.Map{}
	if !util.IsValidEmail(p.Email) {
		data["emailError"] = "Please provide a valid email address"
	}
	if !util.IsValidPassword(p.Password) {
		data["passwordError"] = "Please provide a strong password"
	}
	if len(p.Fullname) < 3 {
		data["fullnameError"] = "Please provide your real full name"
	}
	return data
}

func HandleSignupWithEmail(c *fiber.Ctx) error {
	var params SignupParams
	if err := c.BodyParser(&params); err != nil {
		return err
	}
	if errors := params.Validate(); len(errors) > 0 {
		errors["email"] = params.Email
		errors["fullname"] = params.Fullname
		return flash.WithData(c, errors).Redirect("/signup")
	}
	client := createSupabaseClient()
	resp, err := client.Auth.SignUp(context.Background(), supabase.UserCredentials{
		Email:    params.Email,
		Password: params.Password,
		Data:     fiber.Map{"fullname": params.Fullname},
	})
	if err != nil {
		return err
	}
	logger.Log("msg", "user signup with email", "id", resp.ID)
	return c.Render("auth/email-confirmation", fiber.Map{"email": params.Email})
}

func HandleGetSignin(c *fiber.Ctx) error {
	checkoutID := c.Query("checkoutID")
	return c.Render("auth/signin", fiber.Map{
		"checkoutID": checkoutID,
	})
}

func HandleSigninWithEmail(c *fiber.Ctx) error {
	var credentials supabase.UserCredentials
	if err := c.BodyParser(&credentials); err != nil {
		return err
	}
	client := createSupabaseClient()
	errors := fiber.Map{}
	resp, err := client.Auth.SignIn(context.Background(), credentials)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid login credentials") {
			errors["authError"] = "Invalid credentials, please try again"
		}
		return flash.WithData(c, errors).Redirect("/signin")
	}
	return c.Redirect("/auth/callback/" + resp.AccessToken)
}

func HandleSigninWithGoogle(c *fiber.Ctx) error {
	client := createSupabaseClient()
	resp, err := client.Auth.SignInWithProvider(supabase.ProviderSignInOptions{
		Provider: "google",
	})
	if err != nil {
		return err
	}
	return c.Redirect(resp.URL)
}

func HandleGetSignout(c *fiber.Ctx) error {
	client := createSupabaseClient()
	if err := client.Auth.SignOut(c.Context(), c.Cookies("accessToken")); err != nil {
		return err
	}
	c.ClearCookie("accessToken")
	return c.Redirect("/")
}

// This is the main callback that will be triggered after each authentication (Google or Email).
func HandleAuthCallback(c *fiber.Ctx) error {
	accessToken := c.Params("accessToken")
	if len(accessToken) == 0 {
		return fmt.Errorf("invalid access token")
	}
	c.Cookie(&fiber.Cookie{
		Secure:   true,
		HTTPOnly: true,
		Name:     "accessToken",
		Value:    accessToken,
	})

	client := createSupabaseClient()
	user, err := client.Auth.User(context.Background(), accessToken)
	if err != nil {
		return err
	}
	acc, err := data.CreateAccountForUserIfNotExist(user)
	if err != nil {
		return err
	}

	logger.Log("info", "user signin", "userID", user.ID, "accountID", acc.ID)

	// check if there is a cookie set with a checkout session to redirect the user to
	// when authenticated.
	var (
		checkoutSessionID = c.Cookies("checkoutSessionID")
		redirectTo        = "/domains"
	)
	if len(checkoutSessionID) > 0 {
		stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
		session, err := session.Get(checkoutSessionID, nil)
		if err != nil {
			return err
		}
		// Valid session
		if time.Until(time.Unix(session.ExpiresAt, 0)) > 0 {
			redirectTo = session.URL
		}
		c.Cookie(&fiber.Cookie{
			Name:    "checkoutSessionID",
			Expires: time.Now().AddDate(0, 0, -10),
			Value:   "deleted",
		})
	}

	return c.Redirect(redirectTo)
}

func createSupabaseClient() *supabase.Client {
	return supabase.CreateClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SECRET"), false)
}
