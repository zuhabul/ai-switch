package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type FileVault struct {
	path string
	mu   sync.Mutex
}

type vaultDoc struct {
	Version int               `json:"version"`
	Items   map[string]string `json:"items"` // name -> base64(nonce|ciphertext)
}

func NewFileVault(path string) *FileVault {
	return &FileVault{path: path}
}

func (v *FileVault) Set(name, value string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("secret name is required")
	}
	if value == "" {
		return fmt.Errorf("secret value is required")
	}
	key, err := v.masterKey()
	if err != nil {
		return err
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	doc, err := v.loadUnlocked()
	if err != nil {
		return err
	}
	enc, err := encrypt(key, value)
	if err != nil {
		return err
	}
	doc.Items[name] = enc
	return v.saveUnlocked(doc)
}

func (v *FileVault) Get(name string) (string, error) {
	key, err := v.masterKey()
	if err != nil {
		return "", err
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	doc, err := v.loadUnlocked()
	if err != nil {
		return "", err
	}
	enc, ok := doc.Items[name]
	if !ok {
		return "", fmt.Errorf("secret %q not found", name)
	}
	dec, err := decrypt(key, enc)
	if err != nil {
		return "", err
	}
	return dec, nil
}

func (v *FileVault) Delete(name string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	doc, err := v.loadUnlocked()
	if err != nil {
		return err
	}
	if _, ok := doc.Items[name]; !ok {
		return fmt.Errorf("secret %q not found", name)
	}
	delete(doc.Items, name)
	return v.saveUnlocked(doc)
}

func (v *FileVault) List() ([]string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	doc, err := v.loadUnlocked()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(doc.Items))
	for n := range doc.Items {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

func (v *FileVault) loadUnlocked() (vaultDoc, error) {
	if _, err := os.Stat(v.path); errors.Is(err, os.ErrNotExist) {
		return vaultDoc{Version: 1, Items: map[string]string{}}, nil
	}
	b, err := os.ReadFile(v.path)
	if err != nil {
		return vaultDoc{}, err
	}
	var doc vaultDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return vaultDoc{}, err
	}
	if doc.Items == nil {
		doc.Items = map[string]string{}
	}
	if doc.Version == 0 {
		doc.Version = 1
	}
	return doc, nil
}

func (v *FileVault) saveUnlocked(doc vaultDoc) error {
	if err := os.MkdirAll(filepath.Dir(v.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := v.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, v.path)
}

func (v *FileVault) masterKey() ([]byte, error) {
	if mk := strings.TrimSpace(os.Getenv("AISWITCH_MASTER_KEY")); mk != "" {
		h := sha256.Sum256([]byte(mk))
		return h[:], nil
	}

	h, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	keyPath := filepath.Join(h, ".config", "ai-switch-v2", "master.key")
	if _, err := os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
			return nil, err
		}
		buf := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, buf); err != nil {
			return nil, err
		}
		if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(buf)), 0o600); err != nil {
			return nil, err
		}
		return buf, nil
	}
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	buf, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil {
		return nil, fmt.Errorf("decode master key: %w", err)
	}
	if len(buf) != 32 {
		return nil, fmt.Errorf("invalid master key length: %d", len(buf))
	}
	return buf, nil
}

func encrypt(key []byte, plain string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	joined := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(joined), nil
}

func decrypt(key []byte, enc string) (string, error) {
	blob, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(blob) <= nonceSize {
		return "", fmt.Errorf("invalid encrypted secret payload")
	}
	nonce := blob[:nonceSize]
	ciphertext := blob[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
