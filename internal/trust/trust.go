package trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store represents the trusted projects store
type Store struct {
	Version int              `json:"version"`
	Trusted []TrustedProject `json:"trusted"`
}

// TrustedProject represents a trusted repository
type TrustedProject struct {
	RepoRoot  string    `json:"repo_root"`
	TrustedAt time.Time `json:"trusted_at"`
}

// GetStorePath returns the path to the trusted projects store
func GetStorePath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}

	configDir := filepath.Join(configHome, "sprout")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "trusted-projects.json"), nil
}

// LoadStore loads the trusted projects store
func LoadStore() (*Store, error) {
	storePath, err := GetStorePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No store file yet, return empty store
			return &Store{
				Version: 1,
				Trusted: []TrustedProject{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read trust store: %w", err)
	}

	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse trust store: %w", err)
	}

	return &store, nil
}

// SaveStore saves the trusted projects store
func SaveStore(store *Store) error {
	storePath, err := GetStorePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal trust store: %w", err)
	}

	if err := os.WriteFile(storePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write trust store: %w", err)
	}

	return nil
}

// IsRepoTrusted checks if a repository is trusted
func IsRepoTrusted(repoRoot string) (bool, error) {
	store, err := LoadStore()
	if err != nil {
		return false, err
	}

	for _, project := range store.Trusted {
		if project.RepoRoot == repoRoot {
			return true, nil
		}
	}

	return false, nil
}

// TrustRepo adds a repository to the trusted list
func TrustRepo(repoRoot string) error {
	store, err := LoadStore()
	if err != nil {
		return err
	}

	// Check if already trusted
	for _, project := range store.Trusted {
		if project.RepoRoot == repoRoot {
			return nil // Already trusted
		}
	}

	// Add to trusted list
	store.Trusted = append(store.Trusted, TrustedProject{
		RepoRoot:  repoRoot,
		TrustedAt: time.Now(),
	})

	return SaveStore(store)
}

// UntrustRepo removes a repository from the trusted list
func UntrustRepo(repoRoot string) error {
	store, err := LoadStore()
	if err != nil {
		return err
	}

	// Filter out the repo
	filtered := []TrustedProject{}
	for _, project := range store.Trusted {
		if project.RepoRoot != repoRoot {
			filtered = append(filtered, project)
		}
	}

	store.Trusted = filtered
	return SaveStore(store)
}
