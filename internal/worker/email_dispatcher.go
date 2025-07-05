package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/email"
)

// EmailJob represents an email sending job
type EmailJob struct {
	ID        string
	Email     email.Email
	Retries   int
	CreatedAt time.Time
}

// EmailDispatcher manages email sending workers
type EmailDispatcher struct {
	emailService email.Service
	workers      int
	jobQueue     chan EmailJob
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	logger       *slog.Logger
	maxRetries   int
	retryDelay   time.Duration
}

// Config holds configuration for the email dispatcher
type Config struct {
	Workers     int
	QueueSize   int
	MaxRetries  int
	RetryDelay  time.Duration
	SendTimeout time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Workers:     5,
		QueueSize:   100,
		MaxRetries:  3,
		RetryDelay:  5 * time.Second,
		SendTimeout: 30 * time.Second,
	}
}

// NewEmailDispatcher creates a new email dispatcher
func NewEmailDispatcher(emailService email.Service, config Config, logger *slog.Logger) *EmailDispatcher {
	ctx, cancel := context.WithCancel(context.Background())

	return &EmailDispatcher{
		emailService: emailService,
		workers:      config.Workers,
		jobQueue:     make(chan EmailJob, config.QueueSize),
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger,
		maxRetries:   config.MaxRetries,
		retryDelay:   config.RetryDelay,
	}
}

// Start starts the email dispatcher workers
func (d *EmailDispatcher) Start() {
	d.logger.Info("starting email dispatcher",
		"workers", d.workers,
		"queue_size", cap(d.jobQueue),
	)

	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go d.worker(i)
	}
}

// Stop stops the email dispatcher and waits for workers to finish
func (d *EmailDispatcher) Stop(timeout time.Duration) error {
	d.logger.Info("stopping email dispatcher")

	// Signal workers to stop
	d.cancel()

	// Close the job queue
	close(d.jobQueue)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		d.logger.Info("email dispatcher stopped gracefully")
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for workers to finish")
	}
}

// Enqueue adds an email job to the queue
func (d *EmailDispatcher) Enqueue(email email.Email) error {
	job := EmailJob{
		ID:        generateJobID(),
		Email:     email,
		CreatedAt: time.Now(),
	}

	select {
	case d.jobQueue <- job:
		d.logger.Debug("email job enqueued",
			"job_id", job.ID,
			"to", email.To,
			"subject", email.Subject,
		)
		return nil
	default:
		return fmt.Errorf("email queue is full")
	}
}

// EnqueueWithContext adds an email job to the queue with context
func (d *EmailDispatcher) EnqueueWithContext(ctx context.Context, email email.Email) error {
	job := EmailJob{
		ID:        generateJobID(),
		Email:     email,
		CreatedAt: time.Now(),
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case d.jobQueue <- job:
		d.logger.Debug("email job enqueued",
			"job_id", job.ID,
			"to", email.To,
			"subject", email.Subject,
		)
		return nil
	}
}

// QueueSize returns the current number of jobs in the queue
func (d *EmailDispatcher) QueueSize() int {
	return len(d.jobQueue)
}

// worker processes email jobs
func (d *EmailDispatcher) worker(id int) {
	defer d.wg.Done()

	d.logger.Debug("email worker started", "worker_id", id)

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Debug("email worker stopping", "worker_id", id)
			return
		case job, ok := <-d.jobQueue:
			if !ok {
				d.logger.Debug("email worker stopping (queue closed)", "worker_id", id)
				return
			}

			d.processJob(id, job)
		}
	}
}

// processJob processes a single email job with retries
func (d *EmailDispatcher) processJob(workerID int, job EmailJob) {
	startTime := time.Now()

	d.logger.Debug("processing email job",
		"worker_id", workerID,
		"job_id", job.ID,
		"to", job.Email.To,
		"retries", job.Retries,
	)

	// Create context with timeout for sending
	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	// Try to send the email
	err := d.emailService.Send(ctx, job.Email)

	if err == nil {
		// Success
		d.logger.Info("email sent successfully",
			"worker_id", workerID,
			"job_id", job.ID,
			"to", job.Email.To,
			"duration", time.Since(startTime),
		)
		return
	}

	// Failed
	d.logger.Error("failed to send email",
		"worker_id", workerID,
		"job_id", job.ID,
		"to", job.Email.To,
		"error", err,
		"retries", job.Retries,
	)

	// Check if we should retry
	if job.Retries < d.maxRetries {
		job.Retries++

		// Wait before retry
		select {
		case <-d.ctx.Done():
			return
		case <-time.After(d.retryDelay * time.Duration(job.Retries)):
		}

		// Re-enqueue the job
		select {
		case d.jobQueue <- job:
			d.logger.Debug("email job re-enqueued for retry",
				"job_id", job.ID,
				"retries", job.Retries,
			)
		default:
			d.logger.Error("failed to re-enqueue email job (queue full)",
				"job_id", job.ID,
			)
		}
	} else {
		d.logger.Error("email job failed after max retries",
			"job_id", job.ID,
			"to", job.Email.To,
			"max_retries", d.maxRetries,
		)
	}
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("email-%d-%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

// Stats represents dispatcher statistics
type Stats struct {
	Workers       int
	QueueSize     int
	QueueCapacity int
	Running       bool
}

// GetStats returns current dispatcher statistics
func (d *EmailDispatcher) GetStats() Stats {
	return Stats{
		Workers:       d.workers,
		QueueSize:     len(d.jobQueue),
		QueueCapacity: cap(d.jobQueue),
		Running:       d.ctx.Err() == nil,
	}
}
