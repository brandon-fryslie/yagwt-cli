package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/bmf/yagwt/internal/core"
)

// Store persists workspace metadata
type Store interface {
	// Read operations (lock-free)
	Load() (Metadata, error)
	Get(id string) (WorkspaceMetadata, error)
	FindByName(name string) (WorkspaceMetadata, error)
	FindByPath(path string) (WorkspaceMetadata, error)

	// Write operations (requires lock)
	Save(metadata Metadata) error
	Set(id string, meta WorkspaceMetadata) error
	Delete(id string) error

	// Index operations
	RebuildIndex() error
}

// Metadata is the top-level structure persisted to disk
type Metadata struct {
	SchemaVersion int                          `json:"schemaVersion"`
	Workspaces    map[string]WorkspaceMetadata `json:"workspaces"`
	Index         Index                        `json:"index"`
}

// WorkspaceMetadata is per-workspace persistent data
type WorkspaceMetadata struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Path      string                 `json:"path"`
	Flags     map[string]bool        `json:"flags"`
	Ephemeral *EphemeralMetadata     `json:"ephemeral,omitempty"`
	Activity  ActivityMetadata       `json:"activity"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// EphemeralMetadata contains TTL information
type EphemeralMetadata struct {
	TTLSeconds int       `json:"ttlSeconds"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// ActivityMetadata tracks usage
type ActivityMetadata struct {
	LastOpenedAt      *time.Time `json:"lastOpenedAt,omitempty"`
	LastGitActivityAt *time.Time `json:"lastGitActivityAt,omitempty"`
}

// Index provides reverse lookups
type Index struct {
	ByPath   map[string]string   `json:"byPath"`   // path → ID
	ByName   map[string]string   `json:"byName"`   // name → ID
	ByBranch map[string][]string `json:"byBranch"` // branch → []ID
}

// store implements Store interface
type store struct {
	path string
}

// NewStore creates a new metadata store
func NewStore(gitDir string) (Store, error) {
	storePath := filepath.Join(gitDir, "yagwt", "meta.json")

	// Create directory if it doesn't exist
	storeDir := filepath.Dir(storePath)
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return nil, core.WrapError(core.ErrConfig, "failed to create metadata directory", err).
			WithDetail("path", storeDir)
	}

	return &store{
		path: storePath,
	}, nil
}

// Load reads metadata from disk
func (s *store) Load() (Metadata, error) {
	// Return empty metadata if file doesn't exist
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return Metadata{
			SchemaVersion: 1,
			Workspaces:    make(map[string]WorkspaceMetadata),
			Index: Index{
				ByPath:   make(map[string]string),
				ByName:   make(map[string]string),
				ByBranch: make(map[string][]string),
			},
		}, nil
	}

	// Read file
	data, err := os.ReadFile(s.path)
	if err != nil {
		return Metadata{}, core.WrapError(core.ErrConfig, "failed to read metadata file", err).
			WithDetail("path", s.path)
	}

	// Parse JSON
	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return Metadata{}, core.WrapError(core.ErrConfig, "corrupted metadata file", err).
			WithDetail("path", s.path).
			WithHint("Try removing the metadata file to start fresh", "rm "+s.path)
	}

	// Validate schema version
	if metadata.SchemaVersion != 1 {
		return Metadata{}, core.NewError(core.ErrConfig, "unsupported metadata schema version").
			WithDetail("version", metadata.SchemaVersion).
			WithDetail("expected", 1)
	}

	// Initialize maps if nil
	if metadata.Workspaces == nil {
		metadata.Workspaces = make(map[string]WorkspaceMetadata)
	}
	if metadata.Index.ByPath == nil {
		metadata.Index.ByPath = make(map[string]string)
	}
	if metadata.Index.ByName == nil {
		metadata.Index.ByName = make(map[string]string)
	}
	if metadata.Index.ByBranch == nil {
		metadata.Index.ByBranch = make(map[string][]string)
	}

	return metadata, nil
}

// Get retrieves a workspace by ID
func (s *store) Get(id string) (WorkspaceMetadata, error) {
	metadata, err := s.Load()
	if err != nil {
		return WorkspaceMetadata{}, err
	}

	ws, ok := metadata.Workspaces[id]
	if !ok {
		return WorkspaceMetadata{}, core.NewError(core.ErrNotFound, "workspace not found").
			WithDetail("id", id)
	}

	return ws, nil
}

// FindByName retrieves a workspace by name
func (s *store) FindByName(name string) (WorkspaceMetadata, error) {
	metadata, err := s.Load()
	if err != nil {
		return WorkspaceMetadata{}, err
	}

	id, ok := metadata.Index.ByName[name]
	if !ok {
		return WorkspaceMetadata{}, core.NewError(core.ErrNotFound, "workspace not found").
			WithDetail("name", name)
	}

	ws, ok := metadata.Workspaces[id]
	if !ok {
		// Index is inconsistent
		return WorkspaceMetadata{}, core.NewError(core.ErrBroken, "metadata index is inconsistent").
			WithDetail("name", name).
			WithDetail("id", id).
			WithHint("Try rebuilding the index", "")
	}

	return ws, nil
}

// FindByPath retrieves a workspace by path
func (s *store) FindByPath(path string) (WorkspaceMetadata, error) {
	metadata, err := s.Load()
	if err != nil {
		return WorkspaceMetadata{}, err
	}

	id, ok := metadata.Index.ByPath[path]
	if !ok {
		return WorkspaceMetadata{}, core.NewError(core.ErrNotFound, "workspace not found").
			WithDetail("path", path)
	}

	ws, ok := metadata.Workspaces[id]
	if !ok {
		// Index is inconsistent
		return WorkspaceMetadata{}, core.NewError(core.ErrBroken, "metadata index is inconsistent").
			WithDetail("path", path).
			WithDetail("id", id).
			WithHint("Try rebuilding the index", "")
	}

	return ws, nil
}

// Save writes metadata to disk atomically
func (s *store) Save(metadata Metadata) error {
	// Ensure schema version is set
	metadata.SchemaVersion = 1

	// Marshal to JSON
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return core.WrapError(core.ErrConfig, "failed to marshal metadata", err)
	}

	// Write to temporary file first
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return core.WrapError(core.ErrConfig, "failed to write metadata file", err).
			WithDetail("path", tmpPath)
	}

	// Atomically rename to final path
	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return core.WrapError(core.ErrConfig, "failed to save metadata file", err).
			WithDetail("path", s.path)
	}

	return nil
}

// Set updates a single workspace
func (s *store) Set(id string, meta WorkspaceMetadata) error {
	metadata, err := s.Load()
	if err != nil {
		return err
	}

	// Update workspace
	meta.ID = id
	meta.UpdatedAt = time.Now()
	metadata.Workspaces[id] = meta

	// Update indexes
	s.updateIndexesForWorkspace(&metadata.Index, meta)

	return s.Save(metadata)
}

// Delete removes a workspace
func (s *store) Delete(id string) error {
	metadata, err := s.Load()
	if err != nil {
		return err
	}

	ws, ok := metadata.Workspaces[id]
	if !ok {
		return core.NewError(core.ErrNotFound, "workspace not found").
			WithDetail("id", id)
	}

	// Remove from workspaces
	delete(metadata.Workspaces, id)

	// Clean up indexes
	delete(metadata.Index.ByPath, ws.Path)
	delete(metadata.Index.ByName, ws.Name)

	// Remove from branch index (need to rebuild to clean properly)
	s.rebuildBranchIndex(&metadata)

	return s.Save(metadata)
}

// RebuildIndex rebuilds all indexes from workspace data
func (s *store) RebuildIndex() error {
	metadata, err := s.Load()
	if err != nil {
		return err
	}

	// Clear existing indexes
	metadata.Index = Index{
		ByPath:   make(map[string]string),
		ByName:   make(map[string]string),
		ByBranch: make(map[string][]string),
	}

	// Rebuild from workspaces
	for id, ws := range metadata.Workspaces {
		s.updateIndexesForWorkspace(&metadata.Index, ws)

		// Detect duplicates
		if existingID, ok := metadata.Index.ByPath[ws.Path]; ok && existingID != id {
			// Log warning but continue
			_ = core.NewError(core.ErrBroken, "duplicate path in metadata").
				WithDetail("path", ws.Path).
				WithDetail("id1", existingID).
				WithDetail("id2", id)
		}

		if existingID, ok := metadata.Index.ByName[ws.Name]; ok && existingID != id {
			// Log warning but continue
			_ = core.NewError(core.ErrBroken, "duplicate name in metadata").
				WithDetail("name", ws.Name).
				WithDetail("id1", existingID).
				WithDetail("id2", id)
		}
	}

	return s.Save(metadata)
}

// updateIndexesForWorkspace updates indexes for a single workspace
func (s *store) updateIndexesForWorkspace(index *Index, ws WorkspaceMetadata) {
	// Update path index
	index.ByPath[ws.Path] = ws.ID

	// Update name index
	index.ByName[ws.Name] = ws.ID

	// Note: Branch index would need branch information from workspace
	// For now, we don't have branch info in WorkspaceMetadata
	// This would be populated when integrating with the git wrapper
}

// rebuildBranchIndex rebuilds only the branch index
func (s *store) rebuildBranchIndex(metadata *Metadata) {
	metadata.Index.ByBranch = make(map[string][]string)

	// This would be populated when we integrate with git wrapper
	// to get actual branch information for each workspace
}
