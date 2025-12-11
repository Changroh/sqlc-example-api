-- name: CreateAppointment :one
INSERT INTO appointments (patient_id, doctor_id, time_slot, status, notes)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPatientAppointments :many
SELECT *
FROM appointments
WHERE patient_id = $1
  AND (sqlc.narg(from_ts)::timestamptz IS NULL OR time_slot >= sqlc.narg(from_ts))
  AND (sqlc.narg(to_ts)::timestamptz IS NULL OR time_slot <= sqlc.narg(to_ts))
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY time_slot ASC
LIMIT COALESCE(NULLIF(sqlc.narg(limit_rows), 0), 50)
OFFSET COALESCE(sqlc.narg(offset_rows), 0);

-- name: GetDoctorScheduleBySpecialty :many
SELECT a.*
FROM appointments a
JOIN doctors d ON a.doctor_id = d.id
WHERE d.specialty = $1
  AND (sqlc.narg(from_ts)::timestamptz IS NULL OR a.time_slot >= sqlc.narg(from_ts))
  AND (sqlc.narg(to_ts)::timestamptz IS NULL OR a.time_slot <= sqlc.narg(to_ts))
ORDER BY a.time_slot ASC;

-- name: UpdateAppointmentStatus :one
UPDATE appointments
SET status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

