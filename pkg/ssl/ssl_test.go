package ssl

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/anthdm/ssltracker/data"
	"github.com/anthdm/ssltracker/db"
	"github.com/anthdm/ssltracker/logger"
	"github.com/joho/godotenv"
)

var domains = []string{"taskbrain.io", "certpulse.com", "google.com", "facebook.com", "youtube.com", "twitter.com", "amazon.com", "instagram.com", "linkedin.com", "pinterest.com", "microsoft.com", "apple.com", "netflix.com", "reddit.com", "tumblr.com", "snapchat.com", "wordpress.com", "ebay.com", "whatsapp.com", "dropbox.com", "adobe.com", "yahoo.com", "bing.com", "github.com", "medium.com", "twitch.tv", "stackoverflow.com", "quora.com", "imdb.com", "cnn.com", "bbc.co.uk", "nytimes.com", "wikipedia.org", "huffpost.com", "buzzfeed.com", "etsy.com", "aliexpress.com", "booking.com", "airbnb.com", "tripadvisor.com", "expedia.com", "uber.com", "paypal.com", "visa.com", "mastercard.com", "americanexpress.com", "chase.com", "bankofamerica.com", "wellsfargo.com", "citigroup.com", "hsbc.com", "intel.com", "ibm.com", "oracle.com", "salesforce.com", "cisco.com", "verizon.com", "att.com", "tmobile.com", "sprint.com", "nasa.gov", "spacex.com", "tesla.com", "ford.com", "gm.com", "toyota.com", "bmw.com", "mercedes-benz.com", "nike.com", "adidas.com", "puma.com", "reebok.com", "mcdonalds.com", "starbucks.com", "coke.com", "pepsi.com", "nestle.com", "unilever.com", "coca-cola.com", "pepsiCo.com", "disney.com", "paramount.com", "warnerbros.com", "universalpictures.com", "sony.com", "fox.com", "netflix.com", "hulu.com", "spotify.com", "pandora.com", "soundcloud.com", "booking.com", "agoda.com", "hotels.com", "kayak.com", "expedia.com", "zillow.com", "trulia.com", "realtor.com", "apartments.com", "airbnb.com"}

func TestPollDomain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	info, err := PollDomain(ctx, "sendit.sh")
	if err != nil {
		log.Fatal(err)
	}
	_ = info
}

func TestPollAllDomains(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal(err)
	}
	logger.Init()
	db.Init()

	trackings, err := data.GetAllTrackingsWithAccount()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("got %d trackings\n", len(trackings))

	wg := sync.WaitGroup{}
	for i := 0; i < len(trackings); i++ {
		tracking := trackings[i]
		fmt.Printf("polling domain %s\n", tracking.DomainName)

		wg.Add(1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer func() {
				cancel()
				wg.Done()
			}()
			info, err := PollDomain(ctx, tracking.DomainName)
			if err != nil {
				fmt.Println("failed to poll domain", err)
				return
			}
			fmt.Printf("%s => OK (%d)\n", tracking.DomainName, info.Latency)
		}()
	}

	wg.Wait()
	fmt.Printf("done polling all (%d) domains\n", len(trackings))
}

func TestPoll(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal(err)
	}
	logger.Init()
	db.Init()

	domain := "sprint.com"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	info, err := PollDomain(ctx, domain)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(info)
}

func TestInvalidDomain(t *testing.T) {
	logger.Init()
	wg := sync.WaitGroup{}
	for _, domain := range domains {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			info, err := PollDomain(context.Background(), domain)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("%s => OK | %d\n", domain, info.Latency)
		}(domain)
	}

	wg.Wait()
	fmt.Println("polled", len(domains))
}

func TestInspectNoCert(t *testing.T) {
	resp, err := PollDomain(context.Background(), "sendit.sh")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(resp)
}

func TestGetStatus(t *testing.T) {
	expires := time.Now().AddDate(0, 0, 15)
	status := getStatus(expires)
	if status != "healthy" {
		t.Fatalf("expected status to be healthy got %s", status)
	}
	expires = time.Now().AddDate(0, 0, 14)
	status = getStatus(expires)
	if status != "looming" {
		t.Fatalf("expected status to be looming got %s", status)
	}
	expires = time.Now().AddDate(0, 0, -1)
	status = getStatus(expires)
	if status != "expired" {
		t.Fatalf("expected status to be expired got %s", status)
	}
}
