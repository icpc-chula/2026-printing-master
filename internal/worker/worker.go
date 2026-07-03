package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"printingmaster/internal/models"
)

var ErrNoWorkers = fmt.Errorf("no workers available")

// Pick selects a worker deterministically from the DB-backed worker list
// using jobID % len(workers), per the spec's round-robin assignment.
func Pick(db *gorm.DB, jobID uint) (models.Worker, error) {
	var workers []models.Worker
	if err := db.Order("id asc").Find(&workers).Error; err != nil {
		return models.Worker{}, fmt.Errorf("list workers: %w", err)
	}
	if len(workers) == 0 {
		return models.Worker{}, ErrNoWorkers
	}

	return workers[int(jobID)%len(workers)], nil
}

// baseURL normalizes a worker's stored IP/host into a usable base URL.
func baseURL(ipAddress string) string {
	if strings.HasPrefix(ipAddress, "http://") || strings.HasPrefix(ipAddress, "https://") {
		return strings.TrimSuffix(ipAddress, "/")
	}
	return "http://" + ipAddress
}

// SendPrintJob uploads the given PDF to the worker's /print endpoint as a
// multipart file, matching the FastAPI worker's `UploadFile = File(...)` contract.
func SendPrintJob(ctx context.Context, w models.Worker, filename string, pdf []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(pdf); err != nil {
		return fmt.Errorf("write pdf into form: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	url := baseURL(w.IPAddress) + "/print"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return fmt.Errorf("build worker request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call worker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload struct {
			Message string `json:"message"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(respBody, &payload)
		if payload.Message != "" {
			return fmt.Errorf("worker returned %d: %s", resp.StatusCode, payload.Message)
		}
		return fmt.Errorf("worker returned status %d", resp.StatusCode)
	}

	return nil
}
