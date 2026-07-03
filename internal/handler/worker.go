package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"printingmaster/internal/models"
)

type WorkerHandler struct {
	DB *gorm.DB
}

func NewWorkerHandler(db *gorm.DB) *WorkerHandler {
	return &WorkerHandler{DB: db}
}

type createWorkerRequest struct {
	IPAddress string `json:"ip_address" binding:"required"`
}

// CreateWorker saves a new worker IP address to the database.
func (h *WorkerHandler) CreateWorker(c *gin.Context) {
	var req createWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ip_address is required"})
		return
	}

	ip := strings.TrimSpace(req.IPAddress)
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ip_address is required"})
		return
	}

	w := models.Worker{IPAddress: ip}
	if err := h.DB.Create(&w).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"message": "worker with this ip_address already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save worker"})
		return
	}

	c.JSON(http.StatusCreated, w)
}

// ListWorkers returns all workers currently stored in the database.
func (h *WorkerHandler) ListWorkers(c *gin.Context) {
	var workers []models.Worker
	if err := h.DB.Order("id asc").Find(&workers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list workers"})
		return
	}
	c.JSON(http.StatusOK, workers)
}

// DeleteWorker removes a worker by ID.
func (h *WorkerHandler) DeleteWorker(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid worker id"})
		return
	}

	result := h.DB.Delete(&models.Worker{}, uint(id))
	if result.Error != nil {
		if isForeignKeyViolation(result.Error) {
			c.JSON(http.StatusConflict, gin.H{"message": "worker has existing print jobs and cannot be deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete worker"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "worker not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "worker deleted"})
}

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}

func isForeignKeyViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "foreign key")
}
