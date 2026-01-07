// LifeLogger LL-Journal
// https://api.lifelogger.life
// company: Tellurian Corp (https://www.telluriancorp.com)
// created in: December 2025

package journal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/telluriancorp/ll-journal/internal/git"
	"github.com/telluriancorp/ll-journal/internal/s3"
	"github.com/telluriancorp/ll-journal/internal/store"
)

type Service struct {
	store *store.Store
	s3    *s3.Client
	git   *git.Client
}

func NewService(store *store.Store, s3Client *s3.Client, gitClient *git.Client) *Service {
	return &Service{
		store: store,
		s3:    s3Client,
		git:   gitClient,
	}
}

// CreateJournal creates a new journal
func (s *Service) CreateJournal(ctx context.Context, userSub, title, description string) (store.Journal, error) {
	journal := store.Journal{
		UserSub:     userSub,
		Title:       title,
		Description: sql.NullString{String: description, Valid: description != ""},
	}
	return s.store.CreateJournal(ctx, journal)
}

// GetJournal gets a journal by ID
func (s *Service) GetJournal(ctx context.Context, id, userSub string) (store.Journal, error) {
	return s.store.GetJournal(ctx, id, userSub)
}

// ListJournals lists all journals for a user
func (s *Service) ListJournals(ctx context.Context, userSub string) ([]store.Journal, error) {
	return s.store.ListJournals(ctx, userSub)
}

// UpdateJournal updates a journal
func (s *Service) UpdateJournal(ctx context.Context, id, userSub, title, description string) error {
	journal := store.Journal{
		ID:          id,
		UserSub:     userSub,
		Title:       title,
		Description: sql.NullString{String: description, Valid: description != ""},
	}
	return s.store.UpdateJournal(ctx, journal)
}

// DeleteJournal deletes a journal and all its entries
func (s *Service) DeleteJournal(ctx context.Context, id, userSub string) error {
	// Get all entries first to delete from S3
	entries, err := s.store.ListJournalEntries(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to list entries: %w", err)
	}

	// Delete entries from S3
	for _, entry := range entries {
		entryDate := entry.EntryDate.Format("2006-01-02")
		s3Key := s3.GenerateKey(userSub, id, entryDate)
		if err := s.s3.Delete(ctx, s3Key); err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to delete S3 object %s: %v\n", s3Key, err)
		}
	}

	// Delete from database (cascade will handle entries and versions)
	return s.store.DeleteJournal(ctx, id, userSub)
}

// CreateEntry creates a new journal entry
func (s *Service) CreateEntry(ctx context.Context, userSub, journalID, entryDate, content string) (store.JournalEntry, error) {
	// Validate date format
	date, err := time.Parse("2006-01-02", entryDate)
	if err != nil {
		return store.JournalEntry{}, fmt.Errorf("invalid date format: %w", err)
	}

	// Validate journal exists and belongs to user
	_, err = s.store.GetJournal(ctx, journalID, userSub)
	if err != nil {
		return store.JournalEntry{}, fmt.Errorf("journal not found: %w", err)
	}

	// Check if entry already exists
	_, err = s.store.GetJournalEntryByDate(ctx, journalID, date)
	if err == nil {
		return store.JournalEntry{}, fmt.Errorf("entry for date %s already exists", entryDate)
	}

	// Sanitize content
	content = sanitizeMarkdown(content)

	// Calculate word count
	wordCount := countWords(content)

	// Generate S3 key
	s3Key := s3.GenerateKey(userSub, journalID, entryDate)

	// Upload to S3
	if err := s.s3.Upload(ctx, s3Key, []byte(content)); err != nil {
		return store.JournalEntry{}, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Commit to Git
	commitHash, err := s.git.CommitFile(userSub, journalID, entryDate, content, fmt.Sprintf("Entry for %s", entryDate))
	if err != nil {
		// Try to delete from S3 if Git commit fails
		_ = s.s3.Delete(ctx, s3Key)
		return store.JournalEntry{}, fmt.Errorf("failed to commit to Git: %w", err)
	}

	// Save to database
	entry := store.JournalEntry{
		JournalID:     journalID,
		EntryDate:     date,
		S3Key:         s3Key,
		GitCommitHash: sql.NullString{String: commitHash, Valid: true},
		WordCount:     sql.NullInt32{Int32: int32(wordCount), Valid: true},
	}

	createdEntry, err := s.store.CreateJournalEntry(ctx, entry)
	if err != nil {
		// Try to clean up S3 and Git if database save fails
		_ = s.s3.Delete(ctx, s3Key)
		return store.JournalEntry{}, fmt.Errorf("failed to save entry: %w", err)
	}

	// Save version to database
	version := store.JournalVersion{
		EntryID:       createdEntry.ID,
		CommitHash:    commitHash,
		CommitMessage: sql.NullString{String: fmt.Sprintf("Entry for %s", entryDate), Valid: true},
		AuthorName:    sql.NullString{String: "LifeLogger System", Valid: true},
		AuthorEmail:   sql.NullString{String: "system@lifelogger.life", Valid: true},
		CreatedAt:     time.Now(),
	}
	_, err = s.store.CreateJournalVersion(ctx, version)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to save version: %v\n", err)
	}

	return createdEntry, nil
}

// GetEntry gets a journal entry by date
func (s *Service) GetEntry(ctx context.Context, userSub, journalID, entryDate string) (store.JournalEntry, []byte, error) {
	// Validate date format
	date, err := time.Parse("2006-01-02", entryDate)
	if err != nil {
		return store.JournalEntry{}, nil, fmt.Errorf("invalid date format: %w", err)
	}

	// Get entry from database
	entry, err := s.store.GetJournalEntryByDate(ctx, journalID, date)
	if err != nil {
		return store.JournalEntry{}, nil, fmt.Errorf("entry not found: %w", err)
	}

	// Verify journal belongs to user
	_, err = s.store.GetJournal(ctx, journalID, userSub)
	if err != nil {
		return store.JournalEntry{}, nil, fmt.Errorf("journal not found: %w", err)
	}

	// Download from S3
	content, err := s.s3.Download(ctx, entry.S3Key)
	if err != nil {
		return store.JournalEntry{}, nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return entry, content, nil
}

// UpdateEntry updates an existing journal entry
func (s *Service) UpdateEntry(ctx context.Context, userSub, journalID, entryDate, content string) (store.JournalEntry, error) {
	// Validate date format
	date, err := time.Parse("2006-01-02", entryDate)
	if err != nil {
		return store.JournalEntry{}, fmt.Errorf("invalid date format: %w", err)
	}

	// Get existing entry
	entry, err := s.store.GetJournalEntryByDate(ctx, journalID, date)
	if err != nil {
		return store.JournalEntry{}, fmt.Errorf("entry not found: %w", err)
	}

	// Verify journal belongs to user
	_, err = s.store.GetJournal(ctx, journalID, userSub)
	if err != nil {
		return store.JournalEntry{}, fmt.Errorf("journal not found: %w", err)
	}

	// Sanitize content
	content = sanitizeMarkdown(content)

	// Calculate word count
	wordCount := countWords(content)

	// Upload new version to S3 (overwrite)
	if err := s.s3.Upload(ctx, entry.S3Key, []byte(content)); err != nil {
		return store.JournalEntry{}, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Commit to Git
	commitHash, err := s.git.CommitFile(userSub, journalID, entryDate, content, fmt.Sprintf("Update entry for %s", entryDate))
	if err != nil {
		return store.JournalEntry{}, fmt.Errorf("failed to commit to Git: %w", err)
	}

	// Update database
	entry.GitCommitHash = sql.NullString{String: commitHash, Valid: true}
	entry.WordCount = sql.NullInt32{Int32: int32(wordCount), Valid: true}
	if err := s.store.UpdateJournalEntry(ctx, entry); err != nil {
		return store.JournalEntry{}, fmt.Errorf("failed to update entry: %w", err)
	}

	// Save version to database
	version := store.JournalVersion{
		EntryID:       entry.ID,
		CommitHash:    commitHash,
		CommitMessage: sql.NullString{String: fmt.Sprintf("Update entry for %s", entryDate), Valid: true},
		AuthorName:    sql.NullString{String: "LifeLogger System", Valid: true},
		AuthorEmail:   sql.NullString{String: "system@lifelogger.life", Valid: true},
		CreatedAt:     time.Now(),
	}
	_, err = s.store.CreateJournalVersion(ctx, version)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to save version: %v\n", err)
	}

	return entry, nil
}

// ListEntries lists all entries for a journal
func (s *Service) ListEntries(ctx context.Context, userSub, journalID string) ([]store.JournalEntry, error) {
	// Verify journal belongs to user
	_, err := s.store.GetJournal(ctx, journalID, userSub)
	if err != nil {
		return nil, fmt.Errorf("journal not found: %w", err)
	}

	return s.store.ListJournalEntries(ctx, journalID)
}

// DeleteEntry deletes a journal entry
func (s *Service) DeleteEntry(ctx context.Context, userSub, journalID, entryDate string) error {
	// Validate date format
	date, err := time.Parse("2006-01-02", entryDate)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	// Get entry
	entry, err := s.store.GetJournalEntryByDate(ctx, journalID, date)
	if err != nil {
		return fmt.Errorf("entry not found: %w", err)
	}

	// Verify journal belongs to user
	_, err = s.store.GetJournal(ctx, journalID, userSub)
	if err != nil {
		return fmt.Errorf("journal not found: %w", err)
	}

	// Delete from S3
	if err := s.s3.Delete(ctx, entry.S3Key); err != nil {
		// Log error but continue
		fmt.Printf("Warning: failed to delete S3 object: %v\n", err)
	}

	// Delete from database
	return s.store.DeleteJournalEntry(ctx, entry.ID)
}

// ListVersions lists all versions (commits) for an entry
func (s *Service) ListVersions(ctx context.Context, userSub, journalID, entryDate string) ([]git.CommitInfo, error) {
	// Verify journal belongs to user
	_, err := s.store.GetJournal(ctx, journalID, userSub)
	if err != nil {
		return nil, fmt.Errorf("journal not found: %w", err)
	}

	return s.git.ListCommits(userSub, journalID, entryDate)
}

// GetVersion gets a specific version of an entry
func (s *Service) GetVersion(ctx context.Context, userSub, journalID, entryDate, commitHash string) ([]byte, error) {
	// Verify journal belongs to user
	_, err := s.store.GetJournal(ctx, journalID, userSub)
	if err != nil {
		return nil, fmt.Errorf("journal not found: %w", err)
	}

	return s.git.GetFileContent(userSub, journalID, entryDate, commitHash)
}

// Helper functions

func sanitizeMarkdown(content string) string {
	// Basic sanitization - remove null bytes and normalize line endings
	content = strings.ReplaceAll(content, "\x00", "")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

func countWords(content string) int {
	words := strings.Fields(content)
	return len(words)
}
