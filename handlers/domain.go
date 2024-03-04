package handlers

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anthdm/ssltracker/data"
	"github.com/anthdm/ssltracker/logger"
	"github.com/anthdm/ssltracker/pkg/ssl"
	"github.com/anthdm/ssltracker/settings"
	"github.com/anthdm/ssltracker/util"
	"github.com/gofiber/fiber/v2"
	"github.com/sujit-baniya/flash"
)

var limitFilters = []int{
	5,
	10,
	25,
	50,
	100,
}

var statusFilters = []string{
	"all",
	data.StatusHealthy,
	data.StatusExpires,
	data.StatusExpired,
	data.StatusInvalid,
	data.StatusOffline,
	data.StatusUnresponsive,
}

func HandleDomainList(c *fiber.Ctx) error {
	user := getAuthenticatedUser(c)
	count, err := data.CountUserDomainTrackings(user.ID)
	if err != nil {
		return err
	}
	if count == 0 {
		return c.Render("domains/index", fiber.Map{"userHasTrackings": false})
	}

	filter, err := buildTrackingFilter(c)
	if err != nil {
		return err
	}
	filterContext := buildFilterContext(filter)
	query := fiber.Map{
		"user_id": user.ID,
	}
	if filter.Status != "all" {
		query["status"] = filter.Status
	}
	domainTrackings, err := data.GetDomainTrackings(query, filter.Limit, filter.Page)
	if err != nil {
		return err
	}
	data := fiber.Map{
		"trackings":        domainTrackings,
		"filters":          filterContext,
		"userHasTrackings": true,
		"pages":            buildPages(count, filter.Limit),
		"queryParams":      filter.encode(),
	}
	return c.Render("domains/index", data)
}

func HandleDomainNew(c *fiber.Ctx) error {
	return c.Render("domains/new", fiber.Map{})
}

func HandleDomainDelete(c *fiber.Ctx) error {
	user := getAuthenticatedUser(c)
	query := fiber.Map{
		"user_id": user.ID,
		"id":      c.Params("id"),
	}
	if err := data.DeleteDomainTracking(query); err != nil {
		return err
	}
	return c.Redirect("/domains")
}

func HandleDomainShowRaw(c *fiber.Ctx) error {
	trackingID := c.Params("id")
	user := getAuthenticatedUser(c)
	query := fiber.Map{
		"user_id": user.ID,
		"id":      trackingID,
	}
	tracking, err := data.GetDomainTracking(query)
	if err != nil {
		return err
	}
	return c.Send([]byte(tracking.EncodedPEM))
}

func HandleDomainShow(c *fiber.Ctx) error {
	trackingID := c.Params("id")
	user := getAuthenticatedUser(c)
	query := fiber.Map{
		"user_id": user.ID,
		"id":      trackingID,
	}
	tracking, err := data.GetDomainTracking(query)
	if err != nil {
		return err
	}
	context := fiber.Map{
		"tracking": tracking,
	}
	return c.Render("domains/show", context)
}

func HandleSendTestNotification(c *fiber.Ctx) error {
	time.Sleep(time.Second * 5)
	fmt.Println("sending notification!!!!")
	return c.Send([]byte("notification sent"))
}

func HandleDomainCreate(c *fiber.Ctx) error {
	flashData := fiber.Map{}
	userDomainsInput := c.FormValue("domains")
	userDomainsInput = strings.ReplaceAll(userDomainsInput, " ", "")

	if len(userDomainsInput) == 0 {
		flashData["domainsError"] = "Please provide at least 1 valid domain name"
		return flash.WithData(c, flashData).Redirect("/domains/new")
	}
	domains := strings.Split(userDomainsInput, ",")
	if len(domains) == 0 {
		flashData["domainsError"] = "Invalid domain list input. Make sure to use a comma seperated list (domain1.com, domain2.com, ..)"
		flashData["domains"] = userDomainsInput
		return flash.WithData(c, flashData).Redirect("/domains/new")
	}
	for _, domain := range domains {
		if !util.IsValidDomainName(domain) {
			flashData["domainsError"] = fmt.Sprintf("%s is not a valid domain", domain)
			flashData["domains"] = userDomainsInput
			return flash.WithData(c, flashData).Redirect("/domains/new")
		}
	}

	user := getAuthenticatedUser(c)
	account, err := data.GetUserAccount(user.ID)
	if err != nil {
		return err
	}

	maxTrackings := settings.Account[account.Plan].MaxTrackings
	count, err := data.CountUserDomainTrackings(user.ID)
	if err != nil {
		return err
	}
	if account.Plan > data.PlanStarter && account.SubscriptionStatus != "active" {
		logger.Log("error", "subscription status not active", "status", account.SubscriptionStatus)
		// TODO: ??
		return AppError(fmt.Errorf("subscription status not active"))
		return flash.WithData(c, flashData).Redirect("/domains/new")
	}
	if len(domains)+count > maxTrackings {
		flashData["maxTrackings"] = maxTrackings
		flashData["domains"] = userDomainsInput
		return flash.WithData(c, flashData).Redirect("/domains/new")
	}
	if err := createAllDomainTrackings(user.ID, domains); err != nil {
		return err
	}
	return c.Redirect("/domains")
}

func createAllDomainTrackings(userID string, domains []string) error {
	var (
		trackings = []*data.DomainTracking{}
		wg        = sync.WaitGroup{}
	)
	for _, domain := range domains {
		wg.Add(1)
		go func(domain string) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer func() {
				cancel()
				wg.Done()
			}()
			trackingInfo, err := ssl.PollDomain(ctx, domain)
			if err != nil {
				logger.Log("error", "polling domain failed", "err", err, "domain", domain)
				return
			}
			tracking := &data.DomainTracking{
				DomainName:         domain,
				UserID:             userID,
				DomainTrackingInfo: *trackingInfo,
			}
			trackings = append(trackings, tracking)
		}(domain)
	}
	wg.Wait()

	fmt.Println("inserting domains into the database", len(trackings))

	return data.CreateDomainTrackings(trackings)
}

type TrackingFilter struct {
	Limit  int
	Page   int
	Status string
	Sort   string
}

func (f *TrackingFilter) encode() string {
	values := url.Values{}
	if f.Limit != 0 {
		values.Set("limit", strconv.Itoa(f.Limit))
	}
	if f.Page != 0 {
		values.Set("page", strconv.Itoa(f.Page))
	}
	values.Set("status", f.Status)
	return values.Encode()
}

func buildTrackingFilter(c *fiber.Ctx) (*TrackingFilter, error) {
	filter := new(TrackingFilter)
	if err := c.QueryParser(filter); err != nil {
		return nil, err
	}
	if filter.Limit == 0 {
		filter.Limit = 25
	}
	return filter, nil
}

func buildFilterContext(filter *TrackingFilter) fiber.Map {
	return fiber.Map{
		"statuses":       statusFilters,
		"limits":         limitFilters,
		"selectedStatus": filter.Status,
		"selectedLimit":  filter.Limit,
		"selectedPage":   filter.Page,
	}
}

func buildPages(results int, limit int) []int {
	lenPages := float64(results) / float64(limit)
	pages := make([]int, int(math.RoundToEven(lenPages)))
	for i := 0; i < len(pages); i++ {
		pages[i] = i + 1
	}
	return pages
}
