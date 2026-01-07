// LifeLogger LL-Journal
// https://api.lifelogger.life
// company: Tellurian Corp (https://www.telluriancorp.com)
// created in: December 2025

package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type Store struct {
	db *sql.DB
}

type Journal struct {
	ID          string
	UserSub     string
	Title       string
	Description sql.NullString
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type JournalEntry struct {
	ID            string
	JournalID     string
	EntryDate     time.Time
	S3Key         string
	GitCommitHash sql.NullString
	WordCount     sql.NullInt32
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type JournalVersion struct {
	ID            string
	EntryID       string
	CommitHash    string
	CommitMessage sql.NullString
	AuthorName    sql.NullString
	AuthorEmail   sql.NullString
	CreatedAt     time.Time
}

func New(databaseURL string) (*Store, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// DB returns the underlying database connection
func (s *Store) DB() *sql.DB {
	return s.db
}

// Journal operations

func (s *Store) CreateJournal(ctx context.Context, journal Journal) (Journal, error) {
	if journal.ID == "" {
		journal.ID = generateUUID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO journals (id, user_sub, title, description)
		VALUES ($1, $2, $3, $4)`,
		journal.ID, journal.UserSub, journal.Title, journal.Description)
	if err != nil {
		return Journal{}, err
	}
	return s.GetJournal(ctx, journal.ID, journal.UserSub)
}

func (s *Store) GetJournal(ctx context.Context, id, userSub string) (Journal, error) {
	var journal Journal
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_sub, title, description, created_at, updated_at
		FROM journals
		WHERE id = $1 AND user_sub = $2`,
		id, userSub).Scan(
		&journal.ID, &journal.UserSub, &journal.Title, &journal.Description,
		&journal.CreatedAt, &journal.UpdatedAt)
	if err != nil {
		return Journal{}, err
	}
	return journal, nil
}

func (s *Store) ListJournals(ctx context.Context, userSub string) ([]Journal, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_sub, title, description, created_at, updated_at
		FROM journals
		WHERE user_sub = $1
		ORDER BY created_at DESC`,
		userSub)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var journals []Journal
	for rows.Next() {
		var journal Journal
		if err := rows.Scan(
			&journal.ID, &journal.UserSub, &journal.Title, &journal.Description,
			&journal.CreatedAt, &journal.UpdatedAt); err != nil {
			return nil, err
		}
		journals = append(journals, journal)
	}
	return journals, nil
}

func (s *Store) UpdateJournal(ctx context.Context, journal Journal) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE journals
		SET title = $1, description = $2, updated_at = NOW()
		WHERE id = $3 AND user_sub = $4`,
		journal.Title, journal.Description, journal.ID, journal.UserSub)
	return err
}

func (s *Store) DeleteJournal(ctx context.Context, id, userSub string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM journals
		WHERE id = $1 AND user_sub = $2`,
		id, userSub)
	return err
}

// Journal Entry operations

func (s *Store) CreateJournalEntry(ctx context.Context, entry JournalEntry) (JournalEntry, error) {
	if entry.ID == "" {
		entry.ID = generateUUID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO journal_entries (id, journal_id, entry_date, s3_key, git_commit_hash, word_count)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		entry.ID, entry.JournalID, entry.EntryDate, entry.S3Key, entry.GitCommitHash, entry.WordCount)
	if err != nil {
		return JournalEntry{}, err
	}
	return s.GetJournalEntry(ctx, entry.ID)
}

func (s *Store) GetJournalEntry(ctx context.Context, id string) (JournalEntry, error) {
	var entry JournalEntry
	err := s.db.QueryRowContext(ctx, `
		SELECT id, journal_id, entry_date, s3_key, git_commit_hash, word_count, created_at, updated_at
		FROM journal_entries
		WHERE id = $1`,
		id).Scan(
		&entry.ID, &entry.JournalID, &entry.EntryDate, &entry.S3Key,
		&entry.GitCommitHash, &entry.WordCount, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		return JournalEntry{}, err
	}
	return entry, nil
}

func (s *Store) GetJournalEntryByDate(ctx context.Context, journalID string, entryDate time.Time) (JournalEntry, error) {
	var entry JournalEntry
	dateStr := entryDate.Format("2006-01-02")
	err := s.db.QueryRowContext(ctx, `
		SELECT id, journal_id, entry_date, s3_key, git_commit_hash, word_count, created_at, updated_at
		FROM journal_entries
		WHERE journal_id = $1 AND entry_date = $2`,
		journalID, dateStr).Scan(
		&entry.ID, &entry.JournalID, &entry.EntryDate, &entry.S3Key,
		&entry.GitCommitHash, &entry.WordCount, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		return JournalEntry{}, err
	}
	return entry, nil
}

func (s *Store) ListJournalEntries(ctx context.Context, journalID string) ([]JournalEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, journal_id, entry_date, s3_key, git_commit_hash, word_count, created_at, updated_at
		FROM journal_entries
		WHERE journal_id = $1
		ORDER BY entry_date DESC`,
		journalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []JournalEntry
	for rows.Next() {
		var entry JournalEntry
		if err := rows.Scan(
			&entry.ID, &entry.JournalID, &entry.EntryDate, &entry.S3Key,
			&entry.GitCommitHash, &entry.WordCount, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *Store) UpdateJournalEntry(ctx context.Context, entry JournalEntry) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE journal_entries
		SET s3_key = $1, git_commit_hash = $2, word_count = $3, updated_at = NOW()
		WHERE id = $4`,
		entry.S3Key, entry.GitCommitHash, entry.WordCount, entry.ID)
	return err
}

func (s *Store) DeleteJournalEntry(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM journal_entries
		WHERE id = $1`,
		id)
	return err
}

// Journal Version operations

func (s *Store) CreateJournalVersion(ctx context.Context, version JournalVersion) (JournalVersion, error) {
	if version.ID == "" {
		version.ID = generateUUID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO journal_versions (id, entry_id, commit_hash, commit_message, author_name, author_email, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		version.ID, version.EntryID, version.CommitHash, version.CommitMessage,
		version.AuthorName, version.AuthorEmail, version.CreatedAt)
	if err != nil {
		return JournalVersion{}, err
	}
	return version, nil
}

func (s *Store) ListJournalVersions(ctx context.Context, entryID string) ([]JournalVersion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, entry_id, commit_hash, commit_message, author_name, author_email, created_at
		FROM journal_versions
		WHERE entry_id = $1
		ORDER BY created_at DESC`,
		entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []JournalVersion
	for rows.Next() {
		var version JournalVersion
		if err := rows.Scan(
			&version.ID, &version.EntryID, &version.CommitHash, &version.CommitMessage,
			&version.AuthorName, &version.AuthorEmail, &version.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, nil
}

func (s *Store) GetJournalVersion(ctx context.Context, entryID, commitHash string) (JournalVersion, error) {
	var version JournalVersion
	err := s.db.QueryRowContext(ctx, `
		SELECT id, entry_id, commit_hash, commit_message, author_name, author_email, created_at
		FROM journal_versions
		WHERE entry_id = $1 AND commit_hash = $2`,
		entryID, commitHash).Scan(
		&version.ID, &version.EntryID, &version.CommitHash, &version.CommitMessage,
		&version.AuthorName, &version.AuthorEmail, &version.CreatedAt)
	if err != nil {
		return JournalVersion{}, err
	}
	return version, nil
}

// Helper function to generate UUID
func generateUUID() string {
	// Use timestamp-based ID for now
	// In production, PostgreSQL's gen_random_uuid() is used in the database
	return fmt.Sprintf("journal-%d", time.Now().UnixNano())
}
