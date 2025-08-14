//go:build darwin

package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shaban/macaudio/plugins"
)

// Versions for on-disk files so future migrations can co-exist safely.
const (
	indexVersion   = "1.0-index"
	detailsVersion = "1.0-details"
)

// indexEntry keeps a minimal quick view of a plugin. It's stored in index.json
// and used for fast startup and change detection without full introspection.
type indexEntry struct {
	Key            string    `json:"key"`
	Type           string    `json:"type"`
	Subtype        string    `json:"subtype"`
	ManufacturerID string    `json:"manufacturerID"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	Checksum       string    `json:"checksum"`
	LastSeenAt     time.Time `json:"lastSeenAt"`
}

// indexFile is the on-disk structure for index.json
type indexFile struct {
	Version   string                `json:"version"`
	UpdatedAt time.Time             `json:"updatedAt"`
	Entries   map[string]indexEntry `json:"entries"`
}

// detailsFile is the on-disk structure for a single plugin details file.
// It mirrors the detailed Plugin object and associates it to a checksum of
// the quick info for freshness validation.
type detailsFile struct {
	Version          string          `json:"version"`
	LastIntrospected time.Time       `json:"lastIntrospected"`
	Checksum         string          `json:"checksum"`
	Plugin           *plugins.Plugin `json:"plugin"`
}

func quadKey(t, st, man, name string) string {
	return fmt.Sprintf("%s:%s:%s:%s", t, st, man, name)
}

func checksumQuick(info plugins.PluginInfo) string {
	s := fmt.Sprintf("%s|%s|%s|%s|%s", info.Type, info.Subtype, info.ManufacturerID, info.Name, info.Category)
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// getIndexPaths returns the index.json path and ensures details/ exists.
func getIndexPaths() (string, string, error) {
	dir, err := getPluginCacheDir()
	if err != nil {
		return "", "", err
	}
	detailsDir := filepath.Join(dir, "details")
	if err := os.MkdirAll(detailsDir, 0o755); err != nil {
		return "", "", err
	}
	return filepath.Join(dir, "index.json"), detailsDir, nil
}

// loadIndex reads index.json if present; otherwise returns an empty index.
func loadIndex() (*indexFile, error) {
	idxPath, _, err := getIndexPaths()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(idxPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &indexFile{Version: indexVersion, UpdatedAt: time.Time{}, Entries: map[string]indexEntry{}}, nil
		}
		return nil, err
	}
	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	if idx.Version != indexVersion || idx.Entries == nil {
		return &indexFile{Version: indexVersion, UpdatedAt: time.Time{}, Entries: map[string]indexEntry{}}, nil
	}
	return &idx, nil
}

// saveIndex atomically writes index.json to avoid partial writes.
func saveIndex(idx *indexFile) error {
	idxPath, _, err := getIndexPaths()
	if err != nil {
		return err
	}
	idx.Version = indexVersion
	idx.UpdatedAt = time.Now()
	tmp := idxPath + ".tmp"
	b, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, idxPath)
}

// detailFileName hashes the quadruplet key to a stable filename.
func detailFileName(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:]) + ".json"
}

// readDetails reads a plugin details file and returns the plugin and stored checksum.
func readDetails(key string) (*plugins.Plugin, string, error) {
	_, detailsDir, err := getIndexPaths()
	if err != nil {
		return nil, "", err
	}
	p := filepath.Join(detailsDir, detailFileName(key))
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, "", err
	}
	var df detailsFile
	if err := json.Unmarshal(data, &df); err != nil {
		return nil, "", err
	}
	if df.Version != detailsVersion || df.Plugin == nil {
		return nil, "", fmt.Errorf("invalid details file")
	}
	return df.Plugin, df.Checksum, nil
}

// writeDetails atomically persists a plugin details file.
func writeDetails(key, checksum string, pl *plugins.Plugin) error {
	_, detailsDir, err := getIndexPaths()
	if err != nil {
		return err
	}
	p := filepath.Join(detailsDir, detailFileName(key))
	df := detailsFile{Version: detailsVersion, LastIntrospected: time.Now(), Checksum: checksum, Plugin: pl}
	b, err := json.Marshal(df)
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// deleteDetails removes the cached details file for a given plugin key (best-effort).
func deleteDetails(key string) error {
	_, detailsDir, err := getIndexPaths()
	if err != nil {
		return err
	}
	p := filepath.Join(detailsDir, detailFileName(key))
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
