package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Iknite-Space/sqlc-example-api/db/repo"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	querier repo.Querier
}

func NewHandler(querier repo.Querier) *Handler {
	return &Handler{querier: querier}
}

func (h *Handler) WireHttpHandler() http.Handler {
	r := gin.Default()
	r.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
		c.String(http.StatusInternalServerError, "Internal Server Error: panic")
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	r.POST("/patients", h.handleCreatePatient)
	r.POST("/doctors", h.handleCreateDoctor)
	r.POST("/appointments", h.handleCreateAppointment)
	r.GET("/patients/:id/appointments", h.handleGetPatientAppointments)
	r.GET("/doctors/:specialty/schedule", h.handleGetDoctorSchedule)

	return r
}

type createPatientRequest struct {
	Name      string `json:"name" binding:"required"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	MedicalID string `json:"medical_id"`
}

func (h *Handler) handleCreatePatient(c *gin.Context) {
	var req createPatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	patient, err := h.querier.CreatePatient(c, repo.CreatePatientParams{
		Name:      req.Name,
		Phone:     req.Phone,
		Email:     req.Email,
		MedicalID: req.MedicalID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, patient)
}

type createDoctorRequest struct {
	Name      string `json:"name" binding:"required"`
	Specialty string `json:"specialty" binding:"required"`
	Contact   string `json:"contact"`
}

func (h *Handler) handleCreateDoctor(c *gin.Context) {
	var req createDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doctor, err := h.querier.CreateDoctor(c, repo.CreateDoctorParams{
		Name:      req.Name,
		Specialty: req.Specialty,
		Contact:   req.Contact,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, doctor)
}

type createAppointmentRequest struct {
	PatientID string    `json:"patient_id" binding:"required,uuid"`
	DoctorID  string    `json:"doctor_id" binding:"required,uuid"`
	TimeSlot  time.Time `json:"time_slot" binding:"required"` // ISO8601 expected
	Status    string    `json:"status"`
	Notes     string    `json:"notes"`
}

func (h *Handler) handleCreateAppointment(c *gin.Context) {
	var req createAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TimeSlot.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "time_slot must be in the future"})
		return
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient_id"})
		return
	}
	doctorID, err := uuid.Parse(req.DoctorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid doctor_id"})
		return
	}

	if _, err := h.querier.GetPatientByID(c, patientID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.querier.GetDoctorByID(c, doctorID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "doctor not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := req.Status
	if status == "" {
		status = "confirmed"
	}

	appt, err := h.querier.CreateAppointment(c, repo.CreateAppointmentParams{
		PatientID: patientID,
		DoctorID:  doctorID,
		TimeSlot:  req.TimeSlot,
		Status:    status,
		Notes:     req.Notes,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "doctor already has an appointment at that time_slot"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.enqueueNotifications(c, appt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "appointment created but failed to enqueue notifications: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, appt)
}

func (h *Handler) enqueueNotifications(c *gin.Context, appt repo.Appointment) error {
	now := time.Now()
	confirmAt := now.Add(1 * time.Minute)
	emailAt := appt.TimeSlot.Add(-24 * time.Hour)
	if emailAt.Before(now) {
		emailAt = now
	}

	if _, err := h.querier.CreateNotification(c, repo.CreateNotificationParams{
		AppointmentID: appt.ID,
		Type:          "sms",
		SendAt:        confirmAt,
		Status:        "pending",
	}); err != nil {
		return err
	}
	if _, err := h.querier.CreateNotification(c, repo.CreateNotificationParams{
		AppointmentID: appt.ID,
		Type:          "email",
		SendAt:        emailAt,
		Status:        "pending",
	}); err != nil {
		return err
	}
	return nil
}

func (h *Handler) handleGetPatientAppointments(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient id"})
		return
	}

	from, _ := parseTimePointer(c.Query("from"))
	to, _ := parseTimePointer(c.Query("to"))
	status := c.Query("status")
	limit := parseIntDefault(c.Query("limit"), 50)
	offset := parseIntDefault(c.Query("offset"), 0)

	res, err := h.querier.GetPatientAppointments(c, repo.GetPatientAppointmentsParams{
		PatientID:  patientID,
		FromTs:     nullableTime(from),
		ToTs:       nullableTime(to),
		Status:     status,
		OffsetRows: offset,
		LimitRows:  limit,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusOK, []repo.Appointment{})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) handleGetDoctorSchedule(c *gin.Context) {
	specialty := c.Param("specialty")
	if specialty == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "specialty is required"})
		return
	}

	from, _ := parseTimePointer(c.Query("from"))
	to, _ := parseTimePointer(c.Query("to"))

	res, err := h.querier.GetDoctorScheduleBySpecialty(c, repo.GetDoctorScheduleBySpecialtyParams{
		Specialty: specialty,
		FromTs:    nullableTime(from),
		ToTs:      nullableTime(to),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func parseTimePointer(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func nullableTime(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func parseIntDefault(raw string, def int) int {
	if raw == "" {
		return def
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return val
}
