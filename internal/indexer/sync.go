package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SyncResult represents the results of a sync operation
type SyncResult struct {
	Added   []string
	Updated []string
	Deleted []string
}

// Sync synchronizes the documents directory with the database
// It detects new, updated, and deleted files and updates the index accordingly
func (idx *Indexer) Sync() (*SyncResult, error) {
	fmt.Fprintf(os.Stderr, "[INFO] Starting sync...\n")

	result := &SyncResult{
		Added:   []string{},
		Updated: []string{},
		Deleted: []string{},
	}

	// Step 1: Get files from database (filename -> modified_at)
	dbFiles, err := idx.db.ListDocuments()
	if err != nil {
		return nil, fmt.Errorf("failed to list database files: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[INFO] Found %d documents in database\n", len(dbFiles))

	// Create map for quick lookup
	dbFileMap := make(map[string]time.Time)
	for filename, modTime := range dbFiles {
		dbFileMap[filename] = modTime
	}

	// Step 2: Scan filesystem using document patterns
	fsFiles := make(map[string]time.Time)

	// Get all markdown files matching the configured patterns
	matchedFiles, err := idx.config.GetDocumentFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get document files: %w", err)
	}

	// Get modification times for all matched files
	for _, path := range matchedFiles {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Error accessing %s: %v\n", path, err)
			continue
		}

		// Store file path and modification time
		fsFiles[path] = info.ModTime()
	}

	fmt.Fprintf(os.Stderr, "[INFO] Found %d markdown files in filesystem\n", len(fsFiles))

	// Step 3: Detect changes and process them

	// 3a. Check for new and updated files
	for fsPath, fsMtime := range fsFiles {
		if dbMtime, exists := dbFileMap[fsPath]; !exists {
			// New file: exists in filesystem but not in database
			fmt.Fprintf(os.Stderr, "[INFO] New file detected: %s\n", fsPath)
			result.Added = append(result.Added, fsPath)

			// Index the new file
			if err := idx.IndexFile(fsPath); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to index new file %s: %v\n", fsPath, err)
				// Continue with other files even if one fails
			}
		} else {
			// File exists in both filesystem and database
			// Check if it has been modified by comparing timestamps
			// We need to normalize timestamps to avoid false positives due to precision differences
			if !timeEqual(fsMtime, dbMtime) {
				// Updated file: mtime differs
				fmt.Fprintf(os.Stderr, "[INFO] Updated file detected: %s (fs: %v, db: %v)\n",
					fsPath, fsMtime.Format(time.RFC3339), dbMtime.Format(time.RFC3339))
				result.Updated = append(result.Updated, fsPath)

				// Delete old version from database
				if err := idx.db.DeleteDocument(fsPath); err != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] Failed to delete old version of %s: %v\n", fsPath, err)
					continue
				}

				// Re-index the updated file
				if err := idx.IndexFile(fsPath); err != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] Failed to reindex %s: %v\n", fsPath, err)
					// Continue with other files even if one fails
				}
			}
		}
	}

	// 3b. Check for deleted files
	for dbPath := range dbFileMap {
		if _, exists := fsFiles[dbPath]; !exists {
			// Deleted file: exists in database but not in filesystem
			fmt.Fprintf(os.Stderr, "[INFO] Deleted file detected: %s\n", dbPath)
			result.Deleted = append(result.Deleted, dbPath)

			// Delete from database
			if err := idx.db.DeleteDocument(dbPath); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to delete %s from database: %v\n", dbPath, err)
				// Continue with other files even if one fails
			}
		}
	}

	// Print summary statistics
	fmt.Fprintf(os.Stderr, "[INFO] Sync complete: +%d, ~%d, -%d\n",
		len(result.Added), len(result.Updated), len(result.Deleted))

	return result, nil
}

// timeEqual compares two timestamps with tolerance for filesystem precision differences
// Some filesystems only support second-level precision, while others support nanoseconds
func timeEqual(t1, t2 time.Time) bool {
	// Truncate to second precision to handle filesystem differences
	return t1.Truncate(time.Second).Equal(t2.Truncate(time.Second))
}
