package storage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nutsdb/nutsdb"
	"github.com/thrnjica/agwctl/internal/models"
)

const (
	// Bucket names
	processedAPIsBucket = "processed_apis"
	metadataBucket      = "metadata"

	// Metadata keys
	lastPollKey = "last_poll"
)

// Repository provides data access to the NutsDB database.
type Repository struct {
	db     *nutsdb.DB
	logger *slog.Logger
}

// NewRepository creates a new repository with the specified database path.
func NewRepository(dbPath string, logger *slog.Logger) (*Repository, error) {
	opt := nutsdb.DefaultOptions
	opt.Dir = dbPath
	opt.EntryIdxMode = nutsdb.HintKeyValAndRAMIdxMode

	db, err := nutsdb.Open(opt)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	logger.Info("Database opened", "path", dbPath)

	return &Repository{
		db:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection.
func (r *Repository) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	r.logger.Info("Database closed")
	return nil
}

// IsProcessed checks if an API has been processed.
func (r *Repository) IsProcessed(apiID string) (bool, error) {
	var exists bool

	err := r.db.View(func(tx *nutsdb.Tx) error {
		_, err := tx.Get(processedAPIsBucket, []byte(apiID))
		if err != nil {
			if err == nutsdb.ErrKeyNotFound || err == nutsdb.ErrBucketNotFound {
				exists = false
				return nil
			}
			return err
		}
		exists = true
		return nil
	})

	if err != nil {
		return false, fmt.Errorf("check if processed: %w", err)
	}

	return exists, nil
}

// MarkProcessed marks an API as processed with metadata.
func (r *Repository) MarkProcessed(apiID string, metadata *models.ProcessedAPI) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	err = r.db.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(processedAPIsBucket, []byte(apiID), data, 0)
	})

	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}

	r.logger.Debug("API marked as processed", "api_id", apiID)
	return nil
}

// GetProcessedAPI retrieves metadata for a processed API.
func (r *Repository) GetProcessedAPI(apiID string) (*models.ProcessedAPI, error) {
	var metadata models.ProcessedAPI

	err := r.db.View(func(tx *nutsdb.Tx) error {
		entry, err := tx.Get(processedAPIsBucket, []byte(apiID))
		if err != nil {
			return err
		}

		return json.Unmarshal(entry, &metadata)
	})

	if err != nil {
		if err == nutsdb.ErrKeyNotFound {
			return nil, fmt.Errorf("API not found: %s", apiID)
		}
		return nil, fmt.Errorf("get processed API: %w", err)
	}

	return &metadata, nil
}

// GetAllProcessedIDs retrieves all processed API IDs.
func (r *Repository) GetAllProcessedIDs() ([]string, error) {
	var ids []string

	err := r.db.View(func(tx *nutsdb.Tx) error {
		keys, _, err := tx.GetAll(processedAPIsBucket)
		if err != nil {
			if err == nutsdb.ErrBucketNotFound {
				return nil
			}
			return err
		}

		for _, key := range keys {
			ids = append(ids, string(key))
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("get all processed IDs: %w", err)
	}

	return ids, nil
}

// MarkProcessedBatch marks multiple APIs as processed in a single transaction.
func (r *Repository) MarkProcessedBatch(apis []*models.ProcessedAPI) error {
	err := r.db.Update(func(tx *nutsdb.Tx) error {
		for _, api := range apis {
			data, err := json.Marshal(api)
			if err != nil {
				return fmt.Errorf("marshal API %s: %w", api.ID, err)
			}

			if err := tx.Put(processedAPIsBucket, []byte(api.ID), data, 0); err != nil {
				return fmt.Errorf("put API %s: %w", api.ID, err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("mark processed batch: %w", err)
	}

	r.logger.Debug("Batch marked as processed", "count", len(apis))
	return nil
}

// SetLastPoll stores the timestamp of the last successful poll.
func (r *Repository) SetLastPoll(timestamp time.Time) error {
	data := []byte(timestamp.Format(time.RFC3339))

	err := r.db.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(metadataBucket, []byte(lastPollKey), data, 0)
	})

	if err != nil {
		return fmt.Errorf("set last poll: %w", err)
	}

	return nil
}

// GetLastPoll retrieves the timestamp of the last successful poll.
func (r *Repository) GetLastPoll() (time.Time, error) {
	var timestamp time.Time

	err := r.db.View(func(tx *nutsdb.Tx) error {
		entry, err := tx.Get(metadataBucket, []byte(lastPollKey))
		if err != nil {
			return err
		}

		parsed, err := time.Parse(time.RFC3339, string(entry))
		if err != nil {
			return fmt.Errorf("parse timestamp: %w", err)
		}

		timestamp = parsed
		return nil
	})

	if err != nil {
		if err == nutsdb.ErrKeyNotFound || err == nutsdb.ErrBucketNotFound {
			return time.Time{}, nil // Return zero time if never polled
		}
		return time.Time{}, fmt.Errorf("get last poll: %w", err)
	}

	return timestamp, nil
}

// GetStats returns statistics about the database.
func (r *Repository) GetStats() (map[string]any, error) {
	stats := make(map[string]any)

	err := r.db.View(func(tx *nutsdb.Tx) error {
		// Count processed APIs
		keys, _, err := tx.GetAll(processedAPIsBucket)
		if err != nil && err != nutsdb.ErrBucketNotFound {
			return err
		}
		stats["processed_apis_count"] = len(keys)

		// Get last poll time
		lastPoll, err := r.GetLastPoll()
		if err != nil {
			return err
		}
		if !lastPoll.IsZero() {
			stats["last_poll"] = lastPoll.Format(time.RFC3339)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	return stats, nil
}

// Made with Bob
