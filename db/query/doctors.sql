-- name: CreateDoctor :one
INSERT INTO doctors (name, specialty, contact)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetDoctorByID :one
SELECT * FROM doctors WHERE id = $1;

