// LifeLogger LL-Journal
// https://api.lifelogger.life
// company: Tellurian Corp (https://www.telluriancorp.com)
// created in: December 2025

package git

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Client struct {
	rootDir string
}

type CommitInfo struct {
	Hash        string
	Message     string
	AuthorName  string
	AuthorEmail string
	CreatedAt   time.Time
}

func New(rootDir string) (*Client, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create git root directory: %w", err)
	}
	return &Client{rootDir: rootDir}, nil
}

// GetOrInitRepo gets an existing repository or initializes a new one for a user
func (c *Client) GetOrInitRepo(userSub string) (*git.Repository, error) {
	repoPath := filepath.Join(c.rootDir, userSub)

	// Try to open existing repository
	repo, err := git.PlainOpen(repoPath)
	if err == nil {
		return repo, nil
	}

	// If repository doesn't exist, initialize it
	if err == git.ErrRepositoryNotExists {
		repo, err = git.PlainInit(repoPath, false)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize git repository: %w", err)
		}

		// Create initial commit
		wt, err := repo.Worktree()
		if err != nil {
			return nil, fmt.Errorf("failed to get worktree: %w", err)
		}

		// Create .gitkeep file for initial commit
		gitkeepPath := filepath.Join(repoPath, ".gitkeep")
		if err := os.WriteFile(gitkeepPath, []byte(""), 0644); err != nil {
			return nil, fmt.Errorf("failed to create .gitkeep: %w", err)
		}

		_, err = wt.Add(".gitkeep")
		if err != nil {
			return nil, fmt.Errorf("failed to add .gitkeep: %w", err)
		}

		_, err = wt.Commit("Initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "LifeLogger System",
				Email: "system@lifelogger.life",
				When:  time.Now(),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create initial commit: %w", err)
		}

		return repo, nil
	}

	return nil, fmt.Errorf("failed to open git repository: %w", err)
}

// CommitFile commits a file to the repository
func (c *Client) CommitFile(userSub, journalID, entryDate, content, commitMessage string) (string, error) {
	repo, err := c.GetOrInitRepo(userSub)
	if err != nil {
		return "", err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	// Create directory structure if needed
	repoPath := filepath.Join(c.rootDir, userSub)
	entryDir := filepath.Join(repoPath, journalID)
	if err := os.MkdirAll(entryDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create entry directory: %w", err)
	}

	// Write file
	filePath := filepath.Join(entryDir, fmt.Sprintf("%s.md", entryDate))
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Add file to git
	relativePath := filepath.Join(journalID, fmt.Sprintf("%s.md", entryDate))
	_, err = wt.Add(relativePath)
	if err != nil {
		return "", fmt.Errorf("failed to add file to git: %w", err)
	}

	// Check if there are changes
	status, err := wt.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	if status.IsClean() {
		// No changes, return current HEAD
		ref, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD: %w", err)
		}
		return ref.Hash().String(), nil
	}

	// Create commit
	if commitMessage == "" {
		commitMessage = fmt.Sprintf("Entry for %s", entryDate)
	}

	commit, err := wt.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "LifeLogger System",
			Email: "system@lifelogger.life",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	return commit.String(), nil
}

// GetFileContent gets the content of a file at a specific commit
func (c *Client) GetFileContent(userSub, journalID, entryDate, commitHash string) ([]byte, error) {
	repo, err := c.GetOrInitRepo(userSub)
	if err != nil {
		return nil, err
	}

	var commit *object.Commit
	if commitHash == "" {
		// Get HEAD commit
		ref, err := repo.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD: %w", err)
		}
		commit, err = repo.CommitObject(ref.Hash())
		if err != nil {
			return nil, fmt.Errorf("failed to get commit: %w", err)
		}
	} else {
		// Get specific commit
		hash := plumbing.NewHash(commitHash)
		commit, err = repo.CommitObject(hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit: %w", err)
		}
	}

	// Get file from commit
	filePath := filepath.Join(journalID, fmt.Sprintf("%s.md", entryDate))
	file, err := commit.File(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from commit: %w", err)
	}

	reader, err := file.Reader()
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer reader.Close()

	content := make([]byte, file.Size)
	_, err = reader.Read(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return content, nil
}

// ListCommits lists all commits for a specific file
func (c *Client) ListCommits(userSub, journalID, entryDate string) ([]CommitInfo, error) {
	repo, err := c.GetOrInitRepo(userSub)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(journalID, fmt.Sprintf("%s.md", entryDate))

	// Get HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Iterate through commits
	commits := []CommitInfo{}
	cIter, err := repo.Log(&git.LogOptions{
		From:  ref.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	err = cIter.ForEach(func(commit *object.Commit) error {
		// Check if this commit modified the file
		tree, err := commit.Tree()
		if err != nil {
			return err
		}

		_, err = tree.File(filePath)
		if err == object.ErrFileNotFound {
			// File not in this commit, skip
			return nil
		}
		if err != nil {
			return err
		}

		commits = append(commits, CommitInfo{
			Hash:        commit.Hash.String(),
			Message:     commit.Message,
			AuthorName:  commit.Author.Name,
			AuthorEmail: commit.Author.Email,
			CreatedAt:   commit.Author.When,
		})

		return nil
	})

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}

	return commits, err
}

// GetLatestCommitHash gets the latest commit hash for a file
func (c *Client) GetLatestCommitHash(userSub, journalID, entryDate string) (string, error) {
	repo, err := c.GetOrInitRepo(userSub)
	if err != nil {
		return "", err
	}

	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return ref.Hash().String(), nil
}
