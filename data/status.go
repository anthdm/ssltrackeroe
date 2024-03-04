package data

const (
	StatusHealthy      = "healthy"
	StatusExpires      = "expires"
	StatusExpired      = "expired"
	StatusInvalid      = "invalid"
	StatusOffline      = "offline"
	StatusUnresponsive = "unresponsive"
)

// A list of the Stripe subscription statusses
// active
// past_due
// unpaid
// canceled
// incomplete
// incomplete_expired
// trialing
// paused
const (
	StripeStatusActive            = "active"
	StripeStatusPastDue           = "past_due"
	StripeStatusUnpaid            = "unpaid"
	StripeStatusCanceled          = "canceled"
	StripeStatusIncomplete        = "incomplete"
	StripeStatusIncompleteExpired = "incomplete_expired"
	StripeStatusTrialing          = "trialing"
	StripeStatusPaused            = "paused"
)

func IsPlanActive(s string) bool {
	return s == StripeStatusActive
}
