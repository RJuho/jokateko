// Package service implements the entity mutation workflows shared by the REST API
// and the MCP server: input validation, ID allocation, atomic Markdown persistence,
// store indexing, and live change notification.
package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
)

// Notifier receives entity change events (e.g. "task.updated") for live UI updates.
type Notifier interface {
	Broadcast(eventType string, payload any)
}

// Error kinds returned by Service methods. Use errors.Is to classify an error;
// the error message itself is suitable for end users.
var (
	ErrInvalid  = errors.New("invalid input")
	ErrConflict = errors.New("conflict")
	ErrNotFound = store.ErrNotFound
)

// CodeMilestoneArchived marks the ErrConflict returned when a task is attached to a
// closed or fully completed milestone without the reopen flag.
const CodeMilestoneArchived = "milestone_archived"

type kindError struct {
	kind error
	msg  string
	code string
}

func (e *kindError) Error() string { return e.msg }
func (e *kindError) Unwrap() error { return e.kind }

// ErrorCode returns the machine-readable code of a Service error, or "" when it has none.
func ErrorCode(err error) string {
	if ke, ok := errors.AsType[*kindError](err); ok {
		return ke.code
	}
	return ""
}

func invalidf(format string, args ...any) error {
	return &kindError{kind: ErrInvalid, msg: fmt.Sprintf(format, args...)}
}

func conflictf(format string, args ...any) error {
	return &kindError{kind: ErrConflict, msg: fmt.Sprintf(format, args...)}
}

func notFoundf(format string, args ...any) error {
	return &kindError{kind: ErrNotFound, msg: fmt.Sprintf(format, args...)}
}

// Service performs entity mutations. All mutations are serialized so that
// concurrent read-modify-write cycles from REST and MCP clients cannot lose updates.
type Service struct {
	cfg          *config.Config
	workspaceDir string
	dirs         config.Dirs
	store        *store.Store
	writer       *writer.Writer
	notifier     Notifier

	mu  sync.Mutex
	now func() time.Time
}

// New creates a Service. A nil notifier disables change notifications.
func New(cfg *config.Config, workspaceDir string, st *store.Store, wr *writer.Writer, notifier Notifier) *Service {
	if cfg == nil {
		cfg = config.Default(workspaceDir)
	}
	return &Service{
		cfg:          cfg,
		workspaceDir: workspaceDir,
		dirs:         cfg.ResolveDirs(workspaceDir),
		store:        st,
		writer:       wr,
		notifier:     notifier,
		now:          time.Now,
	}
}

// Config returns the active configuration.
func (s *Service) Config() *config.Config { return s.cfg }

// WorkspaceDir returns the absolute project root directory.
func (s *Service) WorkspaceDir() string { return s.workspaceDir }

// Dirs returns the resolved entity directories.
func (s *Service) Dirs() config.Dirs { return s.dirs }

// Store returns the backing in-memory store, for read-only queries.
func (s *Service) Store() *store.Store { return s.store }

func (s *Service) notify(eventType string, payload any) {
	if s.notifier != nil {
		s.notifier.Broadcast(eventType, payload)
	}
}

func (s *Service) timestamp() string {
	return s.now().UTC().Format(time.RFC3339)
}

// idPattern restricts entity IDs to lowercase slugs so they are always safe to use
// as a single file name inside an entity directory.
var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,99}$`)

// ValidateID reports whether id is an acceptable entity slug.
func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return invalidf("invalid id %q: must be 1-100 characters of lowercase letters, digits and hyphens, starting with a letter or digit", id)
	}
	return nil
}

const maxSlugLen = 40

// Slugify converts a title into a lowercase, hyphen-separated slug of at most
// 40 characters. It returns fallback when the title contains no usable characters.
func Slugify(title, fallback string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			sb.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "-") {
				sb.WriteByte('-')
			}
		}
	}
	res := strings.Trim(sb.String(), "-")
	if len(res) > maxSlugLen {
		res = strings.TrimRight(res[:maxSlugLen], "-")
	}
	if res == "" {
		return fallback
	}
	return res
}

// datedSlug returns a "YYMMDD-slug" ID base used for tasks and milestones.
func (s *Service) datedSlug(title, fallback string) string {
	return s.now().Format("060102") + "-" + Slugify(title, fallback)
}

// allocateID picks the file-system ID for a new entity in dir.
// An explicit ID is validated and must not be taken (ErrConflict).
// A generated ID gets a numeric suffix (-2, -3, ...) until it is free.
// exists reports whether the ID is already indexed in the store.
func allocateID(dir, explicit, generated string, exists func(string) bool) (string, error) {
	taken := func(id string) bool {
		if exists(id) {
			return true
		}
		_, err := os.Lstat(filepath.Join(dir, id+".md"))
		return err == nil
	}

	if explicit != "" {
		if err := ValidateID(explicit); err != nil {
			return "", err
		}
		if taken(explicit) {
			return "", conflictf("an entity with id %q already exists", explicit)
		}
		return explicit, nil
	}

	if err := ValidateID(generated); err != nil {
		return "", err
	}
	for n := 1; n <= 1000; n++ {
		id := generated
		if n > 1 {
			id = fmt.Sprintf("%s-%d", generated, n)
		}
		if !taken(id) {
			return id, nil
		}
	}
	return "", conflictf("could not allocate a free id for %q", generated)
}

// entityPath returns the Markdown path for id in dir, preferring an existing path.
func entityPath(dir, existing, id string) string {
	if existing != "" {
		return existing
	}
	return filepath.Join(dir, id+".md")
}

// CheckTags enforces the controlled tag vocabulary when it is enabled.
func (s *Service) CheckTags(tags []string) error {
	for _, tag := range tags {
		if !s.cfg.IsAllowedTag(tag) {
			return invalidf("tag %q is not permitted. Allowed tags: %v", tag, s.cfg.Tags.Allowed)
		}
	}
	return nil
}

// checkAddedTags runs CheckTags on the tags not already in old, so entities with
// tags that predate the vocabulary stay editable.
func (s *Service) checkAddedTags(old, tags []string) error {
	return s.CheckTags(added(old, tags))
}

// added returns the values in next that are not in prev.
func added(prev, next []string) []string {
	var out []string
	for _, v := range next {
		if !slices.Contains(prev, v) {
			out = append(out, v)
		}
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
