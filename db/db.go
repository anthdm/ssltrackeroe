package db

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

var Bun *bun.DB

func Init() {
	var (
		uri     = fmt.Sprintf("postgres://postgres:%s%s", os.Getenv("DB_PASSWORD"), os.Getenv("DB_URI"))
		sqldb   = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(uri)))
		verbose = false
	)
	Bun = bun.NewDB(sqldb, pgdialect.New())
	if verbose {
		Bun.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}
}

func WhereMap(qb bun.QueryBuilder, m fiber.Map) bun.QueryBuilder {
	for k, v := range m {
		qb = qb.Where(fmt.Sprintf("%s = ?", k), v)
	}
	return qb
}
