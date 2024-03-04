package data

import (
	"context"
	"time"

	"github.com/anthdm/ssltracker/db"
	"github.com/anthdm/ssltracker/logger"
	"github.com/anthdm/ssltracker/util"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

const (
	domainTrackingTable = "domain_trackings"
	defaultLimit        = 25
)

type DomainTrackingInfo struct {
	ServerIP      string
	Issuer        string
	SignatureAlgo string
	PublicKeyAlgo string
	EncodedPEM    string
	PublicKey     string
	Signature     string
	DNSNames      string
	KeyUsage      string
	ExtKeyUsages  []string `bun:",array"`
	Expires       time.Time
	Status        string
	LastPollAt    time.Time
	Latency       int
	Error         string
}

type DomainTracking struct {
	ID         int64 `bun:"id,pk,autoincrement"`
	UserID     string
	DomainName string

	DomainTrackingInfo
}

func CountUserDomainTrackings(userID string) (int, error) {
	return db.Bun.NewSelect().
		Model(&DomainTracking{}).
		Where("user_id = ?", userID).
		Count(context.Background())
}

func GetDomainTrackings(filter fiber.Map, limit int, page int) ([]DomainTracking, error) {
	if limit == 0 {
		limit = defaultLimit
	}
	var trackings []DomainTracking
	builder := db.Bun.NewSelect().Model(&trackings).Limit(limit)
	for k, v := range filter {
		if v != "" {
			builder.Where("? = ?", bun.Ident(k), v)
		}
	}
	offset := (limit - 1) * page
	builder.Offset(offset)
	err := builder.Scan(context.Background())
	return trackings, err
}

func GetDomainTracking(query fiber.Map) (*DomainTracking, error) {
	var (
		tracking = new(DomainTracking)
		ctx      = context.Background()
	)
	builder := db.Bun.NewSelect().Model(tracking).QueryBuilder()
	builder = db.WhereMap(builder, query)
	err := builder.Unwrap().(*bun.SelectQuery).Limit(1).Scan(ctx)
	return tracking, err
}

func DeleteDomainTracking(query fiber.Map) error {
	builder := db.Bun.NewDelete().Model(&DomainTracking{}).QueryBuilder()
	builder = db.WhereMap(builder, query)
	_, err := builder.Unwrap().(*bun.DeleteQuery).Exec(context.Background())
	return err
}

func InsertDomainTracking(tracking *DomainTracking) error {
	_, err := db.Bun.NewInsert().Model(tracking).Exec(context.Background())
	return err
}

func CreateDomainTrackings(trackings []*DomainTracking) (err error) {
	tx, err := db.Bun.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			logger.Log("error", "rollback transaction", "query", "createDomainTrackings", "err", err)
		}
	}()

	for _, tracking := range trackings {
		// Check if already exist. If so, skip.
		query := fiber.Map{
			"domain_name": tracking.DomainName,
			"user_id":     tracking.UserID,
		}
		_, err = GetDomainTracking(query)
		if err != nil {
			if util.IsErrNoRecords(err) {
				if err := InsertDomainTracking(tracking); err != nil {
					return err
				}
			} else {
				logger.Log("error", err)
			}
		}
	}
	return tx.Commit()
}

type TrackingAndAccount struct {
	Account
	DomainTracking
}

func GetAllTrackingsWithAccount() ([]TrackingAndAccount, error) {
	var (
		trackings []TrackingAndAccount
		ctx       = context.Background()
	)
	err := db.Bun.NewSelect().
		ColumnExpr("dt.*").
		ColumnExpr("a.notify_upfront, a.slack_access_token, a.slack_webhook_url").
		TableExpr("domain_trackings as dt").
		Join("INNER JOIN accounts AS a").
		JoinOn("a.user_id = dt.user_id").
		Scan(ctx, &trackings)
	return trackings, err
}

func UpdateAllTrackings(trackings []DomainTracking) error {
	_, err := db.Bun.NewUpdate().
		Model(&trackings).
		Column(
			"issuer",
			"expires",
			"signature_algo",
			"public_key_algo",
			"dns_names",
			"last_poll_at",
			"latency",
			"error",
			"status",
			"signature",
			"public_key",
			"key_usage",
			"ext_key_usages",
			"encoded_pem",
			"server_ip",
		).
		Bulk().
		Exec(context.Background())
	return err
}
