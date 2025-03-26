package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/martin226/slideitin/backend/api/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	StatusQueued     JobStatus = "queued"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

// FirestoreJob is the Firestore representation of a job
// Simplified to contain only essential fields
type FirestoreJob struct {
	ID        string `firestore:"id"`
	Status    string `firestore:"status"`
	Message   string `firestore:"message"`
	CreatedAt int64  `firestore:"createdAt"`
	UpdatedAt int64  `firestore:"updatedAt"`
	ExpiresAt int64  `firestore:"expiresAt,omitempty"`
}

// FirestoreResult is the Firestore representation of a job result
type FirestoreResult struct {
	ID          string `firestore:"id"`
	ResultURL   string `firestore:"resultUrl"`
	PDFData     []byte `firestore:"pdfData"`
	HTMLData    []byte `firestore:"htmlData"`
	CreatedAt   int64  `firestore:"createdAt"`
	ExpiresAt   int64  `firestore:"expiresAt"`
}

// Job represents a single slide generation job with runtime features
type Job struct {
	ID        string
	Theme     string
	Files     []models.File
	Settings  models.SlideSettings
	Status    JobStatus
	Message   string
	ResultURL string
	CreatedAt int64
	UpdatedAt int64
}

// JobUpdate represents an update to a job that can be sent to SSE clients
type JobUpdate struct {
	ID        string    `json:"id"`
	Status    JobStatus `json:"status"`
	Message   string    `json:"message"`
	ResultURL string    `json:"resultUrl,omitempty"`
	UpdatedAt int64     `json:"updatedAt"`
}

// FileReference represents a reference to a file stored locally
type FileReference struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	LocalPath string `json:"localPath"` // Changed from GCSPath
}

// TaskPayload represents the data structure to be sent in a Cloud Task
type TaskPayload struct {
	JobID     string            `json:"jobID"`
	Theme     string            `json:"theme"`
	Files     []FileReference   `json:"files"`
	Settings  models.SlideSettings `json:"settings"`
}

// Service manages jobs using Firestore and direct HTTP calls
type Service struct {
	client     *firestore.Client
	// Removed taskClient, storageClient
	projectID  string
	// Removed region, queueID
	serviceURL string
	// Removed bucketName
	httpClient *http.Client // Add http client
}

// NewService creates a new queue service using Firestore and HTTP client
func NewService(client *firestore.Client) (*Service, error) {
	// Get environment variables
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		// Use a default for local if not set, but it's better if set in .env/docker-compose
		log.Println("Warning: GOOGLE_CLOUD_PROJECT not set, using default 'local-slideitin'")
		projectID = "local-slideitin"
	}

	serviceURL := os.Getenv("SLIDES_SERVICE_URL")
	if serviceURL == "" {
		return nil, fmt.Errorf("SLIDES_SERVICE_URL environment variable is required")
	}

	// Removed Cloud Tasks and Storage client creation
	// Removed region, queueID, bucketName checks

	return &Service{
		client:     client,
		projectID:  projectID,
		serviceURL: serviceURL,
		httpClient: &http.Client{Timeout: time.Second * 30}, // Initialize HTTP client with timeout
	}, nil
}

// Collection returns the Firestore collection reference for jobs
func (s *Service) Collection() *firestore.CollectionRef {
	return s.client.Collection("jobs")
}

// ResultsCollection returns the Firestore collection reference for results
func (s *Service) ResultsCollection() *firestore.CollectionRef {
	return s.client.Collection("results")
}

// saveFileLocally saves a file to the shared volume and returns its local path
func (s *Service) saveFileLocally(ctx context.Context, jobID string, file models.File) (string, error) {
	// Define the directory path within the shared volume
	jobDir := filepath.Join("/shared", jobID)
	// Ensure the directory exists
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create job directory '%s': %v", jobDir, err)
	}

	// Define the full path for the file
	localPath := filepath.Join(jobDir, file.Filename)

	// Write the file data
	if err := os.WriteFile(localPath, file.Data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file to '%s': %v", localPath, err)
	}

	log.Printf("Saved file %s locally to: %s", file.Filename, localPath)
	return localPath, nil
}


// AddJob adds a new job to Firestore, saves files locally, and triggers the slides-service via HTTP
func (s *Service) AddJob(ctx context.Context, id, theme string, fileData []models.File, settings models.SlideSettings) (*Job, error) {
	// Create the job
	now := time.Now().Unix()
	
	// Create a job record for Firestore (simplified)
	firestoreJob := FirestoreJob{
		ID:        id,
		Status:    string(StatusQueued),
		Message:   "Job added to queue",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save to Firestore
	_, err := s.Collection().Doc(id).Set(ctx, firestoreJob)
	if err != nil {
		log.Printf("Failed to add job to Firestore: %v", err)
		return nil, fmt.Errorf("failed to store job: %v", err)
	}

	log.Printf("Added job %s to Firestore", id)

	// Create in-memory job object
	job := &Job{
		ID:        id,
		Theme:     theme,
		Files:     fileData, // Keep original file data here if needed, or clear it
		Settings:  settings,
		Status:    StatusQueued,
		Message:   "Job added to queue",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save files locally to shared volume
	fileRefs := make([]FileReference, 0, len(fileData))
	for _, file := range fileData {
		localPath, err := s.saveFileLocally(ctx, id, file)
		if err != nil {
			// Update job status to failed if file save fails
			s.updateJobStatus(job, StatusFailed, fmt.Sprintf("Failed to save file %s locally: %v", file.Filename, err), "")
			return job, fmt.Errorf("failed to save file locally: %v", err)
		}

		// Create a file reference with the local path
		fileRef := FileReference{
			Filename: file.Filename,
			Type:     file.Type,
			LocalPath: localPath, // Use local path
		}
		fileRefs = append(fileRefs, fileRef)
	}

	// Trigger the slides-service directly via HTTP
	err = s.triggerSlidesService(ctx, job, fileRefs)
	if err != nil {
		// Update job status to failed if triggering fails
		s.updateJobStatus(job, StatusFailed, fmt.Sprintf("Failed to trigger slides service: %v", err), "")
		return job, fmt.Errorf("failed to trigger slides service: %v", err)
	}

	// Optionally update status to processing immediately, or let slides-service do it
	// s.updateJobStatus(job, StatusProcessing, "Sent job to slides service", "")

	return job, nil
}


// triggerSlidesService sends the job details to the slides-service via HTTP POST
func (s *Service) triggerSlidesService(ctx context.Context, job *Job, fileRefs []FileReference) error {
	taskPayload := TaskPayload{
		JobID:    job.ID,
		Theme:    job.Theme,
		Files:    fileRefs, // Contains local paths now
		Settings: job.Settings,
	}

	payloadBytes, err := json.Marshal(taskPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal trigger payload: %v", err)
	}

	// Define the target endpoint
	targetURL := fmt.Sprintf("%s/tasks/process-slides", s.serviceURL) // Use serviceURL from config

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	log.Printf("Triggering slides service for job %s at %s", job.ID, targetURL)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send http request to slides service: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 300 {
		// Try to read body for more info
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slides service returned non-success status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("Successfully triggered slides service for job %s, status: %d", job.ID, resp.StatusCode)
	return nil
}

// GetJob retrieves a job by its ID from Firestore
func (s *Service) GetJob(id string) *Job {
	ctx := context.Background()
	doc, err := s.Collection().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Printf("Job %s not found in Firestore", id)
			return nil
		}
		log.Printf("Error retrieving job %s: %v", id, err)
		return nil
	}

	var firestoreJob FirestoreJob
	if err := doc.DataTo(&firestoreJob); err != nil {
		log.Printf("Error parsing job data: %v", err)
		return nil
	}

	// Check if job has expired
	now := time.Now().Unix()
	if firestoreJob.ExpiresAt > 0 && now > firestoreJob.ExpiresAt {
		// Job has expired, delete it
		_, err := s.Collection().Doc(id).Delete(ctx)
		if err != nil {
			log.Printf("Failed to delete expired job %s: %v", id, err)
		} else {
			log.Printf("Deleted expired job %s", id)
		}
		return nil
	}

	// Get the result if available
	var resultURL string
	if firestoreJob.Status == string(StatusCompleted) {
		resultDoc, err := s.ResultsCollection().Doc(id).Get(ctx)
		if err == nil && resultDoc.Exists() {
			var result FirestoreResult
			if err := resultDoc.DataTo(&result); err == nil {
				resultURL = result.ResultURL
			}
		}
	}

	// Convert to job object
	return &Job{
		ID:        firestoreJob.ID,
		Status:    JobStatus(firestoreJob.Status),
		Message:   firestoreJob.Message,
		ResultURL: resultURL,
		CreatedAt: firestoreJob.CreatedAt,
		UpdatedAt: firestoreJob.UpdatedAt,
	}
}

// WatchJob watches a job for changes and sends updates to the provided channel
// This function will run until the context is canceled or the job reaches a terminal state
func (s *Service) WatchJob(ctx context.Context, jobID string, updates chan<- JobUpdate) error {
	// Get initial job state
	job := s.GetJob(jobID)
	if job == nil {
		return fmt.Errorf("job not found")
	}

	// Send initial status
	updates <- JobUpdate{
		ID:        job.ID,
		Status:    job.Status,
		Message:   job.Message,
		ResultURL: job.ResultURL,
		UpdatedAt: job.UpdatedAt,
	}

	// If job is already in terminal state, we're done
	if job.Status == StatusCompleted || job.Status == StatusFailed {
		close(updates)
		return nil
	}

	// Set up Firestore snapshot listener for real-time updates
	docRef := s.Collection().Doc(jobID)
	snapshots := docRef.Snapshots(ctx)

	// Watch for updates
	for {
		snapshot, err := snapshots.Next()
		if err != nil {
			log.Printf("Error watching job %s: %v", jobID, err)
			return err
		}

		if !snapshot.Exists() {
			log.Printf("Job %s no longer exists", jobID)
			return fmt.Errorf("job deleted")
		}

		var firestoreJob FirestoreJob
		if err := snapshot.DataTo(&firestoreJob); err != nil {
			log.Printf("Error parsing job data: %v", err)
			continue
		}

		// Get result URL if job is completed
		var resultURL string
		if firestoreJob.Status == string(StatusCompleted) {
			resultDoc, err := s.ResultsCollection().Doc(jobID).Get(ctx)
			if err == nil && resultDoc.Exists() {
				var result FirestoreResult
				if err := resultDoc.DataTo(&result); err == nil {
					resultURL = result.ResultURL
				}
			}
		}

		// Send update
		update := JobUpdate{
			ID:        firestoreJob.ID,
			Status:    JobStatus(firestoreJob.Status),
			Message:   firestoreJob.Message,
			ResultURL: resultURL,
			UpdatedAt: firestoreJob.UpdatedAt,
		}

		select {
		case updates <- update:
			// Successfully sent
		case <-ctx.Done():
			// Context was canceled
			return ctx.Err()
		}

		// If job is in terminal state, we're done
		if update.Status == StatusCompleted || update.Status == StatusFailed {
			return nil
		}
	}
}
// updateJobStatus updates a job's status in Firestore
func (s *Service) updateJobStatus(job *Job, status JobStatus, message, resultURL string) {
	ctx := context.Background()
	now := time.Now().Unix()

	// Update job in Firestore
	updates := []firestore.Update{
		{Path: "status", Value: string(status)},
		{Path: "message", Value: message},
		{Path: "updatedAt", Value: now},
	}

	_, err := s.Collection().Doc(job.ID).Update(ctx, updates)
	if err != nil {
		log.Printf("Failed to update job status in Firestore: %v", err)
	}

	// Update the in-memory job
	job.Status = status
	job.Message = message
	job.UpdatedAt = now
	if resultURL != "" {
		job.ResultURL = resultURL
	}

	log.Printf("Job %s updated: status=%s, message=%s", job.ID, status, message)
}

// GetResult retrieves a job result from Firestore
func (s *Service) GetResult(ctx context.Context, jobID string) (*FirestoreResult, error) {
	doc, err := s.ResultsCollection().Doc(jobID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("result not found")
		}
		return nil, fmt.Errorf("error retrieving result: %v", err)
	}
	
	var result FirestoreResult
	if err := doc.DataTo(&result); err != nil {
		return nil, fmt.Errorf("error parsing result data: %v", err)
	}
	
	// Check if result has expired
	now := time.Now().Unix()
	if result.ExpiresAt > 0 && now > result.ExpiresAt {
		// Result has expired, delete it
		_, err := s.ResultsCollection().Doc(jobID).Delete(ctx)
		if err != nil {
			log.Printf("Failed to delete expired result %s: %v", jobID, err)
		} else {
			log.Printf("Deleted expired result %s", jobID)
		}
		return nil, fmt.Errorf("result has expired")
	}
	
	return &result, nil
}
