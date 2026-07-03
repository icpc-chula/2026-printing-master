package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"printingmaster/internal/models"
	"printingmaster/internal/pdfgen"
	"printingmaster/internal/worker"
)

const maxBodyBytes = 32 << 20 // 32MB

type PrintHandler struct {
	DB         *gorm.DB
	StorageDir string
	jobCounter atomic.Uint64
}

func NewPrintHandler(db *gorm.DB, storageDir string) *PrintHandler {
	return &PrintHandler{DB: db, StorageDir: storageDir}
}

func (h *PrintHandler) HandlePrint(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodyBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to read request body"})
		return
	}
	if len(body) > maxBodyBytes {
		c.JSON(http.StatusBadRequest, gin.H{"message": "request body too large"})
		return
	}

	if !utf8.Valid(body) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "body must be valid UTF-8"})
		return
	}

	username := strings.TrimSpace(c.GetHeader("username"))
	teamname := strings.TrimSpace(c.GetHeader("teamname"))
	teamid := strings.TrimSpace(c.GetHeader("teamid"))
	location := strings.TrimSpace(c.GetHeader("location"))

	var missing []string
	if username == "" {
		missing = append(missing, "username")
	}
	if teamname == "" {
		missing = append(missing, "teamname")
	}
	if teamid == "" {
		missing = append(missing, "teamid")
	}
	if location == "" {
		missing = append(missing, "location")
	}
	if len(missing) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("missing required headers: %s", strings.Join(missing, ", ")),
		})
		return
	}

	pdfBytes, err := pdfgen.Generate(string(body), pdfgen.Header{
		Username: username,
		TeamName: teamname,
		TeamID:   teamid,
		Location: location,
	})
	if err != nil {
		log.Printf("pdf generation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate pdf"})
		return
	}

	pages, err := pdfgen.PageCount(pdfBytes)
	if err != nil {
		log.Printf("pdf page count failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to inspect generated pdf"})
		return
	}

	filename := fmt.Sprintf("print_job_%d_%s.pdf", time.Now().UnixNano(), teamname)
	filePath := filepath.Join(h.StorageDir, filename)
	if err := os.MkdirAll(h.StorageDir, 0o755); err != nil {
		log.Printf("failed to create storage dir: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to store generated pdf"})
		return
	}
	if err := os.WriteFile(filePath, pdfBytes, 0o644); err != nil {
		log.Printf("failed to write pdf: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to store generated pdf"})
		return
	}

	jobID := h.jobCounter.Add(1)
	selectedWorker, err := worker.Pick(h.DB, uint(jobID))
	if err != nil {
		log.Printf("worker selection failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "no printing workers available"})
		return
	}

	txn := models.Transaction{
		Username: username,
		TeamName: teamname,
		TeamID:   teamid,
		Location: location,
		FileName: filename,
		FilePath: filePath,
		Pages:    pages,
		WorkerID: selectedWorker.ID,
		Status:   models.StatusPending,
	}
	if err := h.DB.Create(&txn).Error; err != nil {
		log.Printf("failed to persist transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to record print job"})
		return
	}

	if err := worker.SendPrintJob(c.Request.Context(), selectedWorker, filename, pdfBytes); err != nil {
		log.Printf("worker %d failed to print job %d: %v", selectedWorker.ID, txn.ID, err)
		h.markFailed(&txn)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "printing worker failed to print the document"})
		return
	}

	now := time.Now()
	txn.Status = models.StatusPrinted
	txn.PrintedAt = &now
	h.DB.Model(&txn).Updates(map[string]any{
		"status":     models.StatusPrinted,
		"printed_at": now,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":   "print job sent successfully",
		"job_id":    txn.ID,
		"worker_id": selectedWorker.ID,
		"pages":     pages,
		"file_name": filename,
	})
}

func (h *PrintHandler) markFailed(txn *models.Transaction) {
	h.DB.Model(txn).Update("status", models.StatusFailed)
}
