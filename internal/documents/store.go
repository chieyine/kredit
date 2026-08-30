package documents

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScanState string

const (
	ScanPending    ScanState = "PENDING"
	ScanClean      ScanState = "CLEAN"
	ScanRejected   ScanState = "REJECTED"
	ScanQuarantine ScanState = "QUARANTINED"
)

type Document struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	UploadedBy     string    `json:"uploaded_by"`
	Purpose        string    `json:"purpose"`
	ObjectKey      string    `json:"object_key"`
	FileName       string    `json:"file_name"`
	ContentType    string    `json:"content_type"`
	SizeBytes      int64     `json:"size_bytes"`
	SHA256         string    `json:"sha256"`
	ScanState      ScanState `json:"scan_state"`
	RetentionClass string    `json:"retention_class"`
	CreatedAt      time.Time `json:"created_at"`
	ScannedAt      time.Time `json:"scanned_at,omitempty"`
}

type ObjectStore interface {
	Put(context.Context, string, io.Reader, int64, string) error
	SignedURL(context.Context, string, time.Duration) (string, error)
}

type unavailableObjectStore struct{ err error }

func NewUnavailableObjectStore(err error) ObjectStore {
	if err == nil {
		err = errors.New("object storage is unavailable")
	}
	return unavailableObjectStore{err: err}
}
func (s unavailableObjectStore) Put(context.Context, string, io.Reader, int64, string) error {
	return s.err
}
func (s unavailableObjectStore) SignedURL(context.Context, string, time.Duration) (string, error) {
	return "", s.err
}

type UploadSigner interface {
	SignedUploadURL(context.Context, string, time.Duration, string) (string, error)
}

type ObjectMetadataReader interface {
	Head(context.Context, string) (int64, string, error)
}

type Store struct {
	mu      sync.RWMutex
	items   map[string]Document
	objects ObjectStore
	now     func() time.Time
	pool    *pgxpool.Pool
}

func NewStore(objects ObjectStore) *Store {
	return &Store{items: make(map[string]Document), objects: objects, now: func() time.Time { return time.Now().UTC() }}
}

// NewPostgresStore keeps document metadata durable while the object bytes
// remain in the configured private object store. Scan state and upload
// completion are persisted in the same database used by the API and workers.
func NewPostgresStore(pool *pgxpool.Pool, objects ObjectStore) *Store {
	return &Store{items: make(map[string]Document), objects: objects, pool: pool, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Store) Add(ctx context.Context, organizationID, actorID, purpose, fileName, contentType, retentionClass string, size int64, body io.Reader) (Document, error) {
	if strings.TrimSpace(organizationID) == "" || strings.TrimSpace(actorID) == "" || strings.TrimSpace(purpose) == "" {
		return Document{}, errors.New("organization, actor, and purpose are required")
	}
	if size <= 0 || size > 25<<20 {
		return Document{}, errors.New("document size must be between 1 byte and 25 MiB")
	}
	if !allowedType(contentType) {
		return Document{}, errors.New("unsupported document content type")
	}
	if body == nil || s.objects == nil {
		return Document{}, errors.New("document body and object store are required")
	}
	digest := sha256.New()
	reader := io.TeeReader(io.LimitReader(body, size+1), digest)
	key := fmt.Sprintf("%s/%s/%s", organizationID, purpose, newID())
	if err := s.objects.Put(ctx, key, reader, size, contentType); err != nil {
		return Document{}, fmt.Errorf("store document: %w", err)
	}
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return Document{}, fmt.Errorf("read document: %w", err)
	}
	now := s.now()
	doc := Document{ID: newID(), OrganizationID: organizationID, UploadedBy: actorID, Purpose: purpose, ObjectKey: key, FileName: strings.TrimSpace(fileName), ContentType: contentType, SizeBytes: size, SHA256: hex.EncodeToString(digest.Sum(nil)), ScanState: ScanPending, RetentionClass: retentionClass, CreatedAt: now}
	if s.pool != nil {
		if err := s.insert(ctx, doc); err != nil {
			return Document{}, err
		}
		return doc, nil
	}
	s.mu.Lock()
	s.items[doc.ID] = doc
	s.mu.Unlock()
	return doc, nil
}

// CreateUpload creates the quarantined metadata row before the client uploads
// directly to private object storage. The returned URL is intentionally short
// lived and scoped to this single object key.
func (s *Store) CreateUpload(ctx context.Context, organizationID, actorID, purpose, fileName, contentType, retentionClass string, size int64, ttl time.Duration) (Document, string, error) {
	if strings.TrimSpace(organizationID) == "" || strings.TrimSpace(actorID) == "" || strings.TrimSpace(purpose) == "" {
		return Document{}, "", errors.New("organization, actor, and purpose are required")
	}
	if size <= 0 || size > 25<<20 {
		return Document{}, "", errors.New("document size must be between 1 byte and 25 MiB")
	}
	if !allowedType(contentType) {
		return Document{}, "", errors.New("unsupported document content type")
	}
	signer, ok := s.objects.(UploadSigner)
	if !ok {
		return Document{}, "", errors.New("object store does not support direct uploads")
	}
	key := fmt.Sprintf("%s/%s/%s", organizationID, purpose, newID())
	url, err := signer.SignedUploadURL(ctx, key, ttl, contentType)
	if err != nil {
		return Document{}, "", err
	}
	doc := Document{ID: newID(), OrganizationID: organizationID, UploadedBy: actorID, Purpose: purpose, ObjectKey: key, FileName: strings.TrimSpace(fileName), ContentType: contentType, SizeBytes: size, ScanState: ScanPending, RetentionClass: retentionClass, CreatedAt: s.now()}
	if s.pool != nil {
		if err := s.insert(ctx, doc); err != nil {
			return Document{}, "", err
		}
	}
	s.mu.Lock()
	if s.pool == nil {
		s.items[doc.ID] = doc
	}
	s.mu.Unlock()
	return doc, url, nil
}

func (s *Store) CompleteScan(id string, state ScanState) (Document, error) {
	if state != ScanClean && state != ScanRejected && state != ScanQuarantine {
		return Document{}, errors.New("invalid scan state")
	}
	if s.pool != nil {
		var doc Document
		err := s.pool.QueryRow(context.Background(), `UPDATE app.documents SET scan_state = $2, scanned_at = now() WHERE id = $1::uuid RETURNING id::text, COALESCE(organization_id::text,''), uploaded_by::text, purpose, object_key, file_name, content_type, size_bytes, sha256, scan_state, retention_class, created_at, COALESCE(scanned_at, now())`, id, string(state)).Scan(&doc.ID, &doc.OrganizationID, &doc.UploadedBy, &doc.Purpose, &doc.ObjectKey, &doc.FileName, &doc.ContentType, &doc.SizeBytes, &doc.SHA256, &doc.ScanState, &doc.RetentionClass, &doc.CreatedAt, &doc.ScannedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return Document{}, errors.New("document not found")
		}
		if err != nil {
			return Document{}, err
		}
		return doc, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.items[id]
	if !ok {
		return Document{}, errors.New("document not found")
	}
	doc.ScanState = state
	doc.ScannedAt = s.now()
	s.items[id] = doc
	return doc, nil
}

// CompleteUpload verifies that a direct-to-object-storage upload exists and
// matches the declared size before it can enter the scanner queue. It never
// marks a document clean; only a scanner result may do that.
func (s *Store) CompleteUpload(ctx context.Context, id string) (Document, error) {
	doc, ok := s.Get(id)
	if !ok {
		return Document{}, errors.New("document not found")
	}
	reader, ok := s.objects.(ObjectMetadataReader)
	if !ok {
		return Document{}, errors.New("object store does not support upload verification")
	}
	size, contentType, err := reader.Head(ctx, doc.ObjectKey)
	if err != nil {
		return Document{}, fmt.Errorf("verify uploaded document: %w", err)
	}
	if size != doc.SizeBytes || !allowedType(contentType) || !strings.EqualFold(strings.TrimSpace(contentType), strings.TrimSpace(doc.ContentType)) {
		return Document{}, errors.New("uploaded document metadata does not match the declared upload")
	}
	return doc, nil
}

func (s *Store) Get(id string) (Document, bool) {
	if s.pool != nil {
		var doc Document
		err := s.pool.QueryRow(context.Background(), `SELECT id::text, COALESCE(organization_id::text,''), uploaded_by::text, purpose, object_key, file_name, content_type, size_bytes, sha256, scan_state, retention_class, created_at, COALESCE(scanned_at, 'epoch'::timestamptz) FROM app.documents WHERE id = $1::uuid`, id).
			Scan(&doc.ID, &doc.OrganizationID, &doc.UploadedBy, &doc.Purpose, &doc.ObjectKey, &doc.FileName, &doc.ContentType, &doc.SizeBytes, &doc.SHA256, &doc.ScanState, &doc.RetentionClass, &doc.CreatedAt, &doc.ScannedAt)
		return doc, err == nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.items[id]
	return doc, ok
}

func (s *Store) SignedDownload(ctx context.Context, id string, ttl time.Duration) (string, error) {
	doc, ok := s.Get(id)
	if !ok {
		return "", errors.New("document not found")
	}
	if doc.ScanState != ScanClean {
		return "", errors.New("document is not available until scanning is clean")
	}
	return s.objects.SignedURL(ctx, doc.ObjectKey, ttl)
}

func (s *Store) insert(ctx context.Context, doc Document) error {
	if s.pool == nil {
		return errors.New("document database is not configured")
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO app.documents (id, organization_id, uploaded_by, purpose, object_key, file_name, content_type, size_bytes, sha256, scan_state, retention_class, created_at) VALUES ($1::uuid, NULLIF($2,'')::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12)`, doc.ID, doc.OrganizationID, doc.UploadedBy, doc.Purpose, doc.ObjectKey, doc.FileName, doc.ContentType, doc.SizeBytes, doc.SHA256, string(doc.ScanState), doc.RetentionClass, doc.CreatedAt)
	return err
}

func allowedType(contentType string) bool {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "application/pdf", "image/jpeg", "image/png":
		return true
	default:
		return false
	}
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", time.Now().UnixNano()%1_000_000_000_000)
	}
	// UUID-shaped identifiers keep the development and PostgreSQL repositories
	// interchangeable while retaining cryptographic randomness.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

type MemoryObjectStore struct {
	mu      sync.RWMutex
	objects map[string]memoryObject
}

type memoryObject struct {
	data        []byte
	contentType string
}

func NewMemoryObjectStore() *MemoryObjectStore {
	return &MemoryObjectStore{objects: make(map[string]memoryObject)}
}

func (s *MemoryObjectStore) Put(_ context.Context, key string, body io.Reader, size int64, contentType string) error {
	data, err := io.ReadAll(io.LimitReader(body, size+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > size {
		return errors.New("document exceeds declared size")
	}
	s.mu.Lock()
	s.objects[key] = memoryObject{data: append([]byte(nil), data...), contentType: contentType}
	s.mu.Unlock()
	return nil
}

func (s *MemoryObjectStore) SignedURL(_ context.Context, key string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		return "", errors.New("positive URL TTL is required")
	}
	s.mu.RLock()
	_, ok := s.objects[key]
	s.mu.RUnlock()
	if !ok {
		return "", errors.New("object not found")
	}
	return fmt.Sprintf("memory://%s?expires_in=%d", key, int(ttl.Seconds())), nil
}

func (s *MemoryObjectStore) SignedUploadURL(_ context.Context, key string, ttl time.Duration, _ string) (string, error) {
	if ttl <= 0 {
		return "", errors.New("positive URL TTL is required")
	}
	return fmt.Sprintf("memory-upload://%s?expires_in=%d", key, int(ttl.Seconds())), nil
}

func (s *MemoryObjectStore) Head(_ context.Context, key string) (int64, string, error) {
	s.mu.RLock()
	data, ok := s.objects[key]
	s.mu.RUnlock()
	if !ok {
		return 0, "", errors.New("object not found")
	}
	return int64(len(data.data)), data.contentType, nil
}
