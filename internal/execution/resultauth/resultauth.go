package resultauth

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const Domain = "CYBEROS-WORKER-RESULT-V1\n"

type Envelope struct {
	Version           string `json:"version"`
	AssignmentID      string `json:"assignment_id"`
	GrantID           string `json:"grant_id"`
	ProjectID         string `json:"project_id"`
	WorkerID          string `json:"worker_id"`
	HostID            string `json:"host_id"`
	PackContentDigest string `json:"pack_content_digest"`
	ResultHash        string `json:"result_hash"`
	ResultStatus      string `json:"result_status"`
	ResultID          string `json:"result_id"`
	CompletedAt       string `json:"completed_at"`
	Signature         string `json:"signature,omitempty"`
}

func CanonicalBytes(e Envelope) ([]byte, error) {
	m := map[string]string{
		"version": e.Version, "assignment_id": e.AssignmentID, "grant_id": e.GrantID,
		"project_id": e.ProjectID, "worker_id": e.WorkerID, "host_id": e.HostID,
		"pack_content_digest": e.PackContentDigest, "result_hash": e.ResultHash,
		"result_status": e.ResultStatus, "result_id": e.ResultID, "completed_at": e.CompletedAt,
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(m))
	for _, k := range keys {
		ordered[k] = m[k]
	}
	return json.Marshal(ordered)
}

func signedMessage(canon []byte) []byte {
	out := make([]byte, 0, len(Domain)+len(canon))
	out = append(out, []byte(Domain)...)
	out = append(out, canon...)
	return out
}

func HashPayload(status, body string) string {
	h := sha256.Sum256([]byte(status + "\n" + body))
	return hex.EncodeToString(h[:])
}

func Sign(e Envelope, priv ed25519.PrivateKey) (Envelope, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return e, fmt.Errorf("resultauth: invalid private key")
	}
	if e.Version == "" {
		e.Version = "1"
	}
	if e.CompletedAt == "" {
		e.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	}
	canon, err := CanonicalBytes(e)
	if err != nil {
		return e, err
	}
	sig := ed25519.Sign(priv, signedMessage(canon))
	e.Signature = hex.EncodeToString(sig)
	return e, nil
}

func Verify(e Envelope, pub ed25519.PublicKey) error {
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("resultauth: invalid public key")
	}
	if e.Signature == "" {
		return fmt.Errorf("resultauth: missing signature")
	}
	sig, err := hex.DecodeString(e.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("resultauth: malformed signature")
	}
	canon, err := CanonicalBytes(e)
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, signedMessage(canon), sig) {
		return fmt.Errorf("resultauth: invalid signature")
	}
	return nil
}

type KeyRegistry struct {
	mu       sync.RWMutex
	keys     map[string]ed25519.PublicKey
	disabled map[string]bool
}

func NewKeyRegistry() *KeyRegistry {
	return &KeyRegistry{keys: make(map[string]ed25519.PublicKey), disabled: make(map[string]bool)}
}

func (r *KeyRegistry) Register(workerID string, pub ed25519.PublicKey) error {
	if workerID == "" || len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("resultauth: invalid registration")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.keys[workerID]; ok {
		return fmt.Errorf("resultauth: key already registered")
	}
	r.keys[workerID] = pub
	return nil
}

func (r *KeyRegistry) Disable(workerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disabled[workerID] = true
}

func (r *KeyRegistry) Get(workerID string) (ed25519.PublicKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.disabled[workerID] {
		return nil, fmt.Errorf("resultauth: key disabled")
	}
	pub, ok := r.keys[workerID]
	if !ok {
		return nil, fmt.Errorf("resultauth: unknown key")
	}
	return pub, nil
}

type AssignmentBinding struct {
	AssignmentID, GrantID, ProjectID, WorkerID, HostID, PackContentDigest string
}

func VerifyCompletion(e Envelope, expect AssignmentBinding, keys *KeyRegistry) error {
	if e.AssignmentID != expect.AssignmentID || e.GrantID != expect.GrantID {
		return fmt.Errorf("resultauth: assignment/grant mismatch")
	}
	if e.ProjectID != expect.ProjectID || e.WorkerID != expect.WorkerID || e.HostID != expect.HostID {
		return fmt.Errorf("resultauth: identity mismatch")
	}
	if expect.PackContentDigest != "" && e.PackContentDigest != expect.PackContentDigest {
		return fmt.Errorf("resultauth: pack digest mismatch")
	}
	if e.ResultHash == "" || e.ResultID == "" {
		return fmt.Errorf("resultauth: incomplete result")
	}
	pub, err := keys.Get(e.WorkerID)
	if err != nil {
		return err
	}
	return Verify(e, pub)
}

type ReplayGuard struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewReplayGuard() *ReplayGuard {
	return &ReplayGuard{seen: make(map[string]struct{})}
}

func (g *ReplayGuard) Accept(resultID string) error {
	if resultID == "" {
		return fmt.Errorf("resultauth: empty result_id")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	key := strings.TrimSpace(resultID)
	if _, ok := g.seen[key]; ok {
		return fmt.Errorf("resultauth: replay")
	}
	g.seen[key] = struct{}{}
	return nil
}
