package notification

import "os"

var (
	DEFAIULT_SLACK_URL      string = "https://hooks.slack.com/services/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"
	DEFAIULT_SLACK_USERNAME string = "spot-handler-service"
	DEFAIULT_SLACK_CHANNEL  string = "#prod-infra-alerts"
)

func Getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
