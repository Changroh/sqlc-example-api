-- name: CreateNotification :one
INSERT INTO notifications (appointment_id, type, send_at, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListDueNotifications :many
SELECT *
FROM notifications
WHERE status = 'pending'
  AND send_at <= now()
ORDER BY send_at ASC
LIMIT 100;

-- name: MarkNotificationSent :exec
UPDATE notifications
SET status = 'sent',
    attempts = attempts + 1
WHERE id = $1;

-- name: MarkNotificationFailed :exec
UPDATE notifications
SET status = 'failed',
    attempts = attempts + 1,
    last_error = $2
WHERE id = $1;

