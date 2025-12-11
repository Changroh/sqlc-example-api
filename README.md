# Healthcare Appointment & Telemedicine API

This service manages patients, doctors, appointments, and schedules outbound notifications (SMS/email stubs). It uses Go, Gin, PostgreSQL, and sqlc-generated data access.

## Project Structure
- `api/`: HTTP handlers and routing.
- `cmd/api/`: Application entrypoint, config loading, migrations, background notification dispatcher.
- `db/migrations`: Database schema (patients, doctors, appointments, notifications).
- `db/query`: SQL used by sqlc to generate type-safe Go code.
- `db/repo`: Generated sqlc code (do not edit).

## Prerequisites
- Go 1.23+
- PostgreSQL
- `.env` populated (see below)

## Env vars (`.env`)
```
LISTEN_PORT=8085
MIGRATIONS_PATH=./db/migrations
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_HOST=localhost
DB_PORT=5432
DB_Name=messages   # or your DB name
DB_TLS_DISABLED=true

# Notification providers (currently stubbed; add when wiring Twilio/SendGrid)
# TWILIO_ACCOUNT_SID=...
# TWILIO_AUTH_TOKEN=...
# TWILIO_FROM_NUMBER=...
# SENDGRID_API_KEY=...
# EMAIL_FROM=noreply@example.com
```

## Install deps
```
go mod tidy
```

## Generate sqlc code (when queries/migrations change)
```
go generate ./...
```

## Run the API (applies migrations automatically)
```
go run ./cmd/api
```
Server listens on `LISTEN_PORT` (default 8085).

## REST Endpoints
- `POST /patients` — create patient
- `POST /doctors` — create doctor
- `POST /appointments` — create appointment (future time_slot, unique per doctor/time)
- `GET /patients/:id/appointments?from&to&status&limit&offset` — list patient appointments
- `GET /doctors/:specialty/schedule?from&to` — list booked slots for doctors in a specialty

## Example requests (curl)
```bash
# Create patient
curl -X POST http://localhost:8085/patients \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","phone":"+123","email":"john@example.com","medical_id":"MID-1"}'

# Create doctor
curl -X POST http://localhost:8085/doctors \
  -H "Content-Type: application/json" \
  -d '{"name":"Dr Smith","specialty":"cardiology","contact":"+456"}'

# Create appointment (use returned UUIDs)
curl -X POST http://localhost:8085/appointments \
  -H "Content-Type: application/json" \
  -d '{"patient_id":"<patient_uuid>","doctor_id":"<doctor_uuid>","time_slot":"2025-12-12T10:00:00Z","notes":"checkup"}'

# List patient appointments
curl "http://localhost:8085/patients/<patient_uuid>/appointments"

# Doctor schedule by specialty
curl "http://localhost:8085/doctors/cardiology/schedule"
```

## Notifications
A background dispatcher polls `notifications` and marks them sent/failed. SMS/Email sending is stubbed; wire Twilio/SendGrid using the env vars above. Appointments enqueue:
- SMS confirmation at +1 minute
- Email reminder at -24h (or immediately if the window has passed)
