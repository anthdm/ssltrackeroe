package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/anthdm/ssltracker/data"
	"github.com/anthdm/ssltracker/db"
	"github.com/anthdm/ssltracker/logger"
	"github.com/anthdm/ssltracker/pkg/notify"
	"github.com/anthdm/ssltracker/pkg/ssl"
	"github.com/joho/godotenv"
)

type Monitor struct {
	interval time.Duration
	lastPoll time.Time
	quitch   chan struct{}
}

func NewMonitor(interval time.Duration) *Monitor {
	return &Monitor{
		interval: interval,
		quitch:   make(chan struct{}),
	}
}

func (m *Monitor) poll() error {
	trackingsWithAccount, err := data.GetAllTrackingsWithAccount()
	if err != nil {
		return err
	}
	if len(trackingsWithAccount) == 0 {
		logger.Log("msg", "nothing to pulse yet...")
		return nil
	}

	var (
		workers = make(chan struct{}, 15)
		wg      = sync.WaitGroup{}
		results = make(chan data.DomainTracking, len(trackingsWithAccount))
	)
	for _, trackingWithAccount := range trackingsWithAccount {
		wg.Add(1)
		go func(tracking data.TrackingAndAccount) {
			workers <- struct{}{}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer func() {
				<-workers
				wg.Done()
				cancel()
			}()

			domainName := tracking.DomainName
			info, err := ssl.PollDomain(ctx, domainName)
			if err != nil {
				logger.Log("err", err)
				return
			}

			domainTracking := tracking.DomainTracking
			domainTracking.DomainTrackingInfo = *info
			m.maybeNotify(context.Background(), tracking)

			results <- domainTracking
		}(trackingWithAccount)
	}

	wg.Wait()
	close(results)
	return m.processResults(results)
}

func (m *Monitor) maybeNotify(ctx context.Context, tracking data.TrackingAndAccount) error {
	var (
		expires       = tracking.Expires
		notifyUpfront = time.Hour * 24 * time.Duration(tracking.NotifyUpfront)
		account       = tracking.Account
	)
	notifiers := []notify.Notifier{
		notify.NewEmailNotifier([]string{account.NotifyDefaultEmail}),
	}
	if len(account.SlackAccessToken) > 0 {
		notifiers = append(notifiers, notify.NewSlackNotifier(account.SlackWebhookURL))
	}

	for _, notifier := range notifiers {
		if tracking.Status != data.StatusHealthy && tracking.Status != data.StatusExpires {
			c, cancel := context.WithTimeout(ctx, time.Second*2)
			defer cancel()
			if err := notifier.NotifyStatus(c, tracking); err != nil {
				logger.Log("error", err)
			}
		} else if time.Until(expires) <= notifyUpfront {
			c, cancel := context.WithTimeout(ctx, time.Second*2)
			defer cancel()
			if err := notifier.NotifyExpires(c, tracking); err != nil {
				logger.Log("error", err)
			}
		}
	}

	return nil
}

func (m *Monitor) processResults(resultsch chan data.DomainTracking) error {
	var (
		trackings = make([]data.DomainTracking, len(resultsch))
		i         int
	)
	for result := range resultsch {
		trackings[i] = result
		i++
	}
	return data.UpdateAllTrackings(trackings)
}

func (m *Monitor) Start() {
	t := time.NewTicker(m.interval)
	if err := m.poll(); err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case t := <-t.C:
			start := time.Now()
			logger.Log("msg", "new poll", "time", t)
			if err := m.poll(); err != nil {
				logger.Log("error", "monitor poll error", "err", err)
			}
			logger.Log("msg", "poll complete", "took", time.Since(start))
		case <-m.quitch:
			logger.Log("msg", "monitor quitting...", "lastPoll", m.lastPoll)
			return
		}
	}
}

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal(err)
	}
	db.Init()
	logger.Init()

	m := NewMonitor(time.Second * 10)
	m.Start()
}
