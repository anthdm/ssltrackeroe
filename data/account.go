package data

import (
	"context"

	"github.com/anthdm/ssltracker/db"
	"github.com/anthdm/ssltracker/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/nedpals/supabase-go"
	"github.com/uptrace/bun"
)

type Plan int

func (p Plan) String() string {
	switch p {
	case PlanStarter:
		return "starter"
	case PlanBusiness:
		return "business"
	case PlanEnterprise:
		return "enterprise"
	default:
		return "unknown"
	}
}

const (
	PlanStarter Plan = iota
	PlanBusiness
	PlanEnterprise
)

type Account struct {
	ID                   int64 `bun:",pk,autoincrement"`
	UserID               string
	StripeCustomerID     string
	StripeSubscriptionID string
	SubscriptionStatus   string
	Plan                 Plan
	NotifyUpfront        int
	NotifyDefaultEmail   string
	NotifyWebhookURL     string
	SlackAccessToken     string
	SlackChannelID       string
	SlackWebhookURL      string
}

func GetUserAccount(userID string) (*Account, error) {
	account := new(Account)
	ctx := context.Background()
	err := db.Bun.NewSelect().
		Model(account).
		Where("user_id = ?", userID).
		Scan(ctx)
	return account, err
}

func GetAccount(query fiber.Map) (*Account, error) {
	account := new(Account)
	builder := db.Bun.NewSelect().Model(account)
	for k, v := range query {
		builder.Where("? = ?", bun.Ident(k), v)
	}
	err := builder.Scan(context.Background())
	return account, err
}

func UpdateAccount(acc *Account) error {
	_, err := db.Bun.NewUpdate().
		Model(acc).
		WherePK().
		Exec(context.Background())
	return err
}

func CreateAccountForUserIfNotExist(user *supabase.User) (*Account, error) {
	if acc, err := GetUserAccount(user.ID); err == nil {
		return acc, nil
	}
	acc := Account{
		UserID:             user.ID,
		NotifyUpfront:      7,
		NotifyDefaultEmail: user.Email,
		Plan:               PlanStarter,
	}
	_, err := db.Bun.NewInsert().Model(&acc).Exec(context.Background())
	if err != nil {
		return nil, err
	}
	logger.Log("event", "new account signup", "id", acc.ID)
	return &acc, nil
}
