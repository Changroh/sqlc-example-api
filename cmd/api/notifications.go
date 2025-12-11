package main

import (
	"context"
	"log"
	"time"

	"github.com/Iknite-Space/sqlc-example-api/db/repo"
)

// startNotificationDispatcher polls for due notifications and marks them sent/failed.
func startNotificationDispatcher(ctx context.Context, q repo.Querier, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := dispatchOnce(ctx, q); err != nil {
					log.Printf("notification dispatcher error: %v", err)
				}
			}
		}
	}()
}

func dispatchOnce(ctx context.Context, q repo.Querier) error {
	items, err := q.ListDueNotifications(ctx)
	if err != nil {
		return err
	}

	for _, n := range items {
		if err := simulateSend(ctx, n); err != nil {
			_ = q.MarkNotificationFailed(ctx, repo.MarkNotificationFailedParams{
				ID:        n.ID,
				LastError: err.Error(),
			})
			continue
		}
		_ = q.MarkNotificationSent(ctx, n.ID)
	}
	return nil
}

// simulateSend stands in for Twilio/SendGrid; replace with real clients later.
func simulateSend(_ context.Context, n repo.Notification) error {
	log.Printf("sending %s notification for appointment %s (send_at=%s)", n.Type, n.AppointmentID, n.SendAt)
	return nil
}
