package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthdm/ssltracker/data"
	"github.com/anthdm/ssltracker/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/checkout/session"
	"github.com/stripe/stripe-go/v74/price"
	"github.com/stripe/stripe-go/v74/product"
	"github.com/stripe/stripe-go/v74/webhook"
)

func HandleStripeCheckoutCreate(c *fiber.Ctx) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	priceID := c.FormValue("priceID")
	if len(priceID) == 0 {
		return fmt.Errorf("invalid price id")
	}
	successCallback := os.Getenv("STRIPE_CHECKOUT_SUCCESS_CALLBACK")
	cancelCallback := os.Getenv("STRIPE_CHECKOUT_CANCEL_CALLBACK")
	params := &stripe.CheckoutSessionParams{
		SuccessURL: &successCallback,
		CancelURL:  &cancelCallback,
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	s, err := session.New(params)
	if err != nil {
		return err
	}

	if !isUserSignedIn(c) {
		c.Cookie(&fiber.Cookie{
			Secure:   true,
			HTTPOnly: true,
			Name:     "checkoutSessionID",
			Value:    s.ID,
		})
		return c.Redirect("/signin")
	}

	return c.Redirect(s.URL)
}

func HandleStripeCheckoutSuccess(c *fiber.Ctx) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	sessionID := c.Query("sessionID")
	session, err := session.Get(sessionID, nil)
	if err != nil {
		return err
	}
	// The checkout session may be complete but the payment processing may still be in process.
	// Available checkout session statuses (open, complete, expired)
	user := getAuthenticatedUser(c)
	account, err := data.GetUserAccount(user.ID)
	if err != nil {
		return err
	}
	account.StripeCustomerID = session.Customer.ID
	account.StripeSubscriptionID = session.Subscription.ID
	if err := data.UpdateAccount(account); err != nil {
		return err
	}
	return c.Render("checkout/success", fiber.Map{})
}

// After the subscription signup succeeds, the customer returns to your website at the success_url,
// which initiates a checkout.session.completed webhooks. When you receive a checkout.session.completed event,
// you can provision the subscription. Continue to provision each month (if billing monthly) as you receive invoice.paid events.
// If you receive an invoice.payment_failed event, notify your customer and send them to the customer portal to update their payment method.
func HandleStripeWebhook(c *fiber.Ctx) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	var (
		body          = c.Request().Body()
		sigHeader     = c.GetReqHeaders()["Stripe-Signature"]
		webhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
	)
	event, err := webhook.ConstructEvent(body, sigHeader, webhookSecret)
	if err != nil {
		return err
	}
	switch event.Type {
	case "customer.subscription.deleted":
		var sub stripe.Subscription
		b, _ := event.Data.Raw.MarshalJSON()
		if err := json.Unmarshal(b, &sub); err != nil {
			return err
		}
		account, err := data.GetAccount(fiber.Map{"stripe_subscription_id": sub.ID})
		if err != nil {
			return err
		}
		logger.Log("msg", "subscription cancelled", "id", sub.ID, "accountID", account.ID)

	case "customer.subscription.updated":
		// Hack so we dont receive this hook before the success page was triggered storing
		// all the stripe account information in the database so we can find the account right here.
		time.Sleep(time.Second * 4)
		var sub stripe.Subscription
		b, _ := event.Data.Raw.MarshalJSON()
		if err := json.Unmarshal(b, &sub); err != nil {
			return err
		}
		query := fiber.Map{
			"stripe_subscription_id": sub.ID,
		}
		account, err := data.GetAccount(query)
		if err != nil {
			return err
		}
		account.SubscriptionStatus = string(sub.Status)
		priceID := sub.Items.Data[0].Price.ID
		sprice, err := price.Get(priceID, nil)
		if err != nil {
			return err
		}
		product, err := product.Get(sprice.Product.ID, nil)
		if err != nil {
			return err
		}
		plan := planFromProductName(product.Name)
		account.Plan = plan
		if err := data.UpdateAccount(account); err != nil {
			return err
		}
		logger.Log("msg", "updated customer subscription", "subscription", sub.ID, "status", sub.Status, "plan", plan)
	}
	return nil
}

func HandleStripeCheckoutCancel(c *fiber.Ctx) error {
	return nil
}

func planFromProductName(name string) data.Plan {
	switch strings.ToLower(name) {
	case "business":
		return data.PlanBusiness
	case "enterprise":
		return data.PlanEnterprise
	default:
		return data.PlanStarter
	}
}
