// Copyright (c) 2026 IBM (https://ibm.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package store provides persistent storage for API processing state.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nutsdb/nutsdb"

	"github.com/thrnjica/agwctl/internal/models"
)

const (
	// Bucket names
	bucketDone     = "processed_apis"
	bucketMetadata = "metadata"

	// Metadata keys
	lastPollKey = "last_poll"
)

// Store provides data access to the NutsDB database.
type Store struct {
	ndb *nutsdb.DB
	log *slog.Logger
}

// New creates a new repository with the specified database path.
func New(dir string, log *slog.Logger) (*Store, error) {
	opt := nutsdb.DefaultOptions
	opt.Dir = dir
	opt.EntryIdxMode = nutsdb.HintKeyValAndRAMIdxMode

	ndb, err := nutsdb.Open(opt)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	log.Info("Database opened", slog.String("path", dir))

	return &Store{
		ndb: ndb,
		log: log,
	}, nil
}

// Close closes the database connection.
func (r *Store) Close() error {
	if err := r.ndb.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	r.log.Info("Database closed")
	return nil
}

// Processed checks if an API has been processed by id.
func (r *Store) Processed(id string) (bool, error) {
	var exists bool

	err := r.ndb.View(func(tx *nutsdb.Tx) error {
		_, err := tx.Get(bucketDone, []byte(id))
		if err != nil {
			if errors.Is(err, nutsdb.ErrKeyNotFound) ||
				errors.Is(err, nutsdb.ErrBucketNotFound) {
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
func (r *Store) MarkProcessed(id string, meta *models.API) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	err = r.ndb.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(bucketDone, []byte(id), data, 0)
	})
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}

	r.log.Debug("API marked as processed", slog.String("api_id", id))
	return nil
}

// Get retrieves metadata for a processed API.
func (r *Store) Get(id string) (*models.API, error) {
	var meta models.API

	err := r.ndb.View(func(tx *nutsdb.Tx) error {
		entry, err := tx.Get(bucketDone, []byte(id))
		if err != nil {
			return err
		}
		return json.Unmarshal(entry, &meta)
	})
	if err != nil {
		if errors.Is(err, nutsdb.ErrKeyNotFound) {
			return nil, fmt.Errorf("unknown API: %s", id)
		}
		return nil, fmt.Errorf("get processed API: %w", err)
	}

	return &meta, nil
}

// IDs retrieves all processed API IDs.
func (r *Store) IDs() ([]string, error) {
	var ids []string

	err := r.ndb.View(func(tx *nutsdb.Tx) error {
		keys, _, err := tx.GetAll(bucketDone)
		if err != nil {
			if errors.Is(err, nutsdb.ErrBucketNotFound) {
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
func (r *Store) MarkProcessedBatch(apis []*models.API) error {
	err := r.ndb.Update(func(tx *nutsdb.Tx) error {
		for _, api := range apis {
			data, err := json.Marshal(api)
			if err != nil {
				return fmt.Errorf("marshal API %s: %w", api.ID, err)
			}

			if err := tx.Put(bucketDone, []byte(api.ID), data, 0); err != nil {
				return fmt.Errorf("put API %s: %w", api.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("mark processed batch: %w", err)
	}

	r.log.Debug("Batch marked as processed", slog.Int("count", len(apis)))
	return nil
}

// UpdateLastPoll stores the timestamp of the last successful poll.
func (r *Store) UpdateLastPoll(ts time.Time) error {
	data := []byte(ts.Format(time.RFC3339))

	err := r.ndb.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(bucketMetadata, []byte(lastPollKey), data, 0)
	})
	if err != nil {
		return fmt.Errorf("set last poll: %w", err)
	}

	return nil
}

// LastPoll retrieves the timestamp of the last successful poll.
func (r *Store) LastPoll() (time.Time, error) {
	var ts time.Time

	err := r.ndb.View(func(tx *nutsdb.Tx) error {
		entry, err := tx.Get(bucketMetadata, []byte(lastPollKey))
		if err != nil {
			return err
		}

		parsed, err := time.Parse(time.RFC3339, string(entry))
		if err != nil {
			return fmt.Errorf("parse timestamp: %w", err)
		}

		ts = parsed
		return nil
	})
	if err != nil {
		if errors.Is(err, nutsdb.ErrKeyNotFound) ||
			errors.Is(err, nutsdb.ErrBucketNotFound) {
			return time.Time{}, nil // Return zero time if never polled
		}
		return time.Time{}, fmt.Errorf("get last poll: %w", err)
	}

	return ts, nil
}

// Stats returns statistics about the database.
func (r *Store) Stats() (map[string]any, error) {
	stats := make(map[string]any)

	err := r.ndb.View(func(tx *nutsdb.Tx) error {
		// Count processed APIs
		keys, _, err := tx.GetAll(bucketDone)
		if err != nil && !errors.Is(err, nutsdb.ErrBucketNotFound) {
			return err
		}
		stats["processed_apis_count"] = len(keys)

		// Get last poll time
		lastPoll, err := r.LastPoll()
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
