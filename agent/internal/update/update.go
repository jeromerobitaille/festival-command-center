// Package update : mise à jour automatique de l'agent depuis les releases GitHub.
//
// Chaîne de confiance : checksums.txt est signé en CI (ed25519) ; l'agent vérifie la signature avec
// la clé publique embarquée, puis vérifie le SHA-256 du binaire téléchargé avant de le mettre en place.
package update

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

//go:embed public_key.txt
var publicKeyB64 string

// DefaultBaseURL : page des releases du dépôt (sans slash final).
const DefaultBaseURL = "https://github.com/jeromerobitaille/festival-command-center/releases"

var ErrUpToDate = errors.New("déjà à jour")

type Options struct {
	BaseURL string
	Current string // version courante ("0.1.0"), "dev" = build local jamais mis à jour sauf Force
	Force   bool
	ExePath string
	Client  *http.Client
	Logf    func(string, ...any)
}

type Release struct {
	Version string
	Asset   string
	SHA256  string
	URL     string
}

func PublicKey() (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKeyB64))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("clé publique de release embarquée invalide")
	}
	return ed25519.PublicKey(raw), nil
}

func (o *Options) defaults() {
	if o.BaseURL == "" {
		o.BaseURL = DefaultBaseURL
	}
	o.BaseURL = strings.TrimRight(o.BaseURL, "/")
	if o.Client == nil {
		o.Client = &http.Client{Timeout: 5 * time.Minute}
	}
	if o.Logf == nil {
		o.Logf = func(string, ...any) {}
	}
	if o.ExePath == "" {
		o.ExePath, _ = os.Executable()
	}
}

// Check récupère checksums.txt de la dernière release, vérifie sa signature et retourne la release
// applicable à cette plateforme si elle est plus récente que Current.
func Check(ctx context.Context, o Options) (*Release, error) {
	o.defaults()
	pub, err := PublicKey()
	if err != nil {
		return nil, err
	}
	sums, err := fetch(ctx, o.Client, o.BaseURL+"/latest/download/checksums.txt", 1<<20)
	if err != nil {
		return nil, fmt.Errorf("checksums.txt : %w", err)
	}
	sigB64, err := fetch(ctx, o.Client, o.BaseURL+"/latest/download/checksums.txt.sig", 4096)
	if err != nil {
		return nil, fmt.Errorf("checksums.txt.sig : %w", err)
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigB64)))
	if err != nil || !ed25519.Verify(pub, sums, sig) {
		return nil, fmt.Errorf("signature de checksums.txt INVALIDE : mise à jour refusée")
	}

	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	re := regexp.MustCompile(`^agent_(\d+\.\d+\.\d+[0-9A-Za-z.\-]*)_` + runtime.GOOS + `_` + runtime.GOARCH + regexp.QuoteMeta(suffix) + `$`)
	var rel *Release
	sc := bufio.NewScanner(strings.NewReader(string(sums)))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) != 2 {
			continue
		}
		if m := re.FindStringSubmatch(f[1]); m != nil {
			rel = &Release{Version: m[1], Asset: f[1], SHA256: strings.ToLower(f[0])}
		}
	}
	if rel == nil {
		return nil, fmt.Errorf("aucun binaire pour %s/%s dans la dernière release", runtime.GOOS, runtime.GOARCH)
	}
	rel.URL = o.BaseURL + "/download/v" + rel.Version + "/" + rel.Asset
	if !o.Force {
		if o.Current == "dev" || o.Current == "" {
			return rel, fmt.Errorf("%w : build de développement (%s), utiliser -force", ErrUpToDate, o.Current)
		}
		if semver.Compare("v"+rel.Version, "v"+o.Current) <= 0 {
			return rel, ErrUpToDate
		}
	}
	return rel, nil
}

// Apply télécharge le binaire, vérifie son SHA-256 et remplace l'exécutable courant.
// L'ancien binaire est conservé en <exe>.old (voir CleanupOld).
func Apply(ctx context.Context, o Options, rel *Release) error {
	o.defaults()
	dir := filepath.Dir(o.ExePath)
	tmp := filepath.Join(dir, ".agent.download")
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return fmt.Errorf("écriture dans %s : %w", dir, err)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, rel.URL, nil)
	resp, err := o.Client.Do(req)
	if err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("téléchargement %s : %s", rel.URL, resp.Status)
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, 300<<20))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != rel.SHA256 {
		os.Remove(tmp)
		return fmt.Errorf("SHA-256 du binaire incorrect (attendu %s, reçu %s) : mise à jour refusée", rel.SHA256[:12], got[:12])
	}
	o.Logf("[update] %s téléchargé (%d octets), hash vérifié", rel.Asset, n)

	old := o.ExePath + ".old"
	os.Remove(old)
	if err := os.Rename(o.ExePath, old); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renommage de l'ancien binaire : %w", err)
	}
	if err := os.Rename(tmp, o.ExePath); err != nil {
		os.Rename(old, o.ExePath) // retour arrière
		return fmt.Errorf("mise en place du nouveau binaire : %w", err)
	}
	return nil
}

// CleanupOld supprime le binaire précédent conservé après une mise à jour.
func CleanupOld(exePath string) {
	if exePath == "" {
		exePath, _ = os.Executable()
	}
	os.Remove(exePath + ".old")
}

func fetch(ctx context.Context, c *http.Client, url string, limit int64) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s : %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}
