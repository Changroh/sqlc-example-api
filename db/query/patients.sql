-- name: CreatePatient :one
INSERT INTO patients (name, phone, email, medical_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPatientByID :one
SELECT * FROM patients WHERE id = $1;

