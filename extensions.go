package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------
// SPARROW EXTENSIONS
//
// An extension is a .zip package (kept in the app store / on disk).
//   - Install : the zip is validated and stored in  extensions/packages/<id>.zip
//   - Enable  : the zip is EXTRACTED to             extensions/installed/<id>/
//               and its scripts/styles start running on matching pages
//   - Disable : the extracted folder is deleted (the zip stays)
//   - Remove  : disable + delete the zip
//
// manifest.json (at the zip root, or inside a single top folder):
//   {
//     "id": "video-speed-controller",       required, [a-z0-9._-]
//     "name": "Video Speed Controller",     required
//     "version": "1.0.0",
//     "description": "...",
//     "author": "Codone",
//     "icon": "⏩",                         emoji (optional)
//     "matches": ["<all_urls>"],            chrome-style match patterns
//     "exclude_matches": [],
//     "run_at": "document_idle",            document_start|document_end|document_idle
//     "js": ["content.js"],
//     "css": ["style.css"]
//   }
// ---------------------------------------------------

const (
	extMaxZipBytes      = 20 << 20
	extMaxFiles         = 500
	extMaxFileBytes     = 8 << 20
	extMaxTotalBytes    = 40 << 20
	extManifestFileName = "manifest.json"
)

var extIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,62}$`)

type ExtManifest struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	Author         string   `json:"author"`
	Icon           string   `json:"icon"`
	Matches        []string `json:"matches"`
	ExcludeMatches []string `json:"exclude_matches"`
	RunAt          string   `json:"run_at"`
	JS             []string `json:"js"`
	CSS            []string `json:"css"`
	// HostPermissions lists the https hosts the extension may fetch through
	// sparrowExt.fetch (Chrome-style patterns, e.g. "https://api.example.com/*").
	HostPermissions []string `json:"host_permissions"`
}

type ExtInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Icon        string   `json:"icon"`
	Matches     []string `json:"matches"`
	Enabled     bool     `json:"enabled"`
	SizeKB      int      `json:"sizeKB"`
}

type loadedExt struct {
	manifest ExtManifest
	js       string
	css      string
	include  []*regexp.Regexp
	exclude  []*regexp.Regexp
	hosts    []*regexp.Regexp
}

type ExtensionManager struct {
	mu      sync.Mutex
	root    string
	enabled map[string]bool
	loaded  map[string]*loadedExt
}

type extState struct {
	Enabled []string `json:"enabled"`
}

// extUIToken protects the management bindings: only the sparrow://extensions
// page (which receives the token when it is served) can install/enable/remove.
var extUIToken = randomHexToken(16)

func randomHexToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func defaultExtensionsRoot() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "Sparrow", "extensions")
}

var extensions = newExtensionManager(defaultExtensionsRoot())

func newExtensionManager(root string) *ExtensionManager {
	return &ExtensionManager{
		root:    root,
		enabled: make(map[string]bool),
		loaded:  make(map[string]*loadedExt),
	}
}

func (m *ExtensionManager) packagesDir() string  { return filepath.Join(m.root, "packages") }
func (m *ExtensionManager) installedDir() string { return filepath.Join(m.root, "installed") }
func (m *ExtensionManager) statePath() string    { return filepath.Join(m.root, "extensions.json") }

func (m *ExtensionManager) packagePath(id string) string {
	return filepath.Join(m.packagesDir(), id+".zip")
}

func (m *ExtensionManager) installedPath(id string) string {
	return filepath.Join(m.installedDir(), id)
}

// Load restores the enabled set from disk and (re)extracts what is needed.
func (m *ExtensionManager) Load() {
	m.mu.Lock()
	defer m.mu.Unlock()

	_ = os.MkdirAll(m.packagesDir(), 0o755)
	_ = os.MkdirAll(m.installedDir(), 0o755)

	var st extState
	if raw, err := os.ReadFile(m.statePath()); err == nil {
		_ = json.Unmarshal(raw, &st)
	}
	m.enabled = make(map[string]bool)
	m.loaded = make(map[string]*loadedExt)
	for _, id := range st.Enabled {
		if !extIDPattern.MatchString(id) {
			continue
		}
		if _, err := os.Stat(m.packagePath(id)); err != nil {
			continue // package vanished
		}
		if _, err := os.Stat(filepath.Join(m.installedPath(id), extManifestFileName)); err != nil {
			if err := m.extractLocked(id); err != nil {
				continue
			}
		}
		ext, err := loadExtFromDir(m.installedPath(id))
		if err != nil {
			continue
		}
		m.enabled[id] = true
		m.loaded[id] = ext
	}
	m.saveStateLocked()
}

func (m *ExtensionManager) saveStateLocked() {
	ids := make([]string, 0, len(m.enabled))
	for id, on := range m.enabled {
		if on {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	raw, err := json.MarshalIndent(extState{Enabled: ids}, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(m.root, 0o755)
	_ = os.WriteFile(m.statePath(), raw, 0o644)
}

// ---------- reading a package ----------

// readPackage validates a zip and returns its manifest and the prefix
// (folder) the manifest lives in.
func readPackage(data []byte) (*ExtManifest, string, *zip.Reader, error) {
	if len(data) == 0 {
		return nil, "", nil, errors.New("empty file")
	}
	if len(data) > extMaxZipBytes {
		return nil, "", nil, errors.New("extension package is too large (max 20 MB)")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, "", nil, errors.New("not a valid zip file")
	}
	if len(zr.File) > extMaxFiles {
		return nil, "", nil, errors.New("too many files in package")
	}

	var total uint64
	for _, f := range zr.File {
		total += f.UncompressedSize64
		if f.UncompressedSize64 > extMaxFileBytes {
			return nil, "", nil, fmt.Errorf("file %q is too large", f.Name)
		}
	}
	if total > extMaxTotalBytes {
		return nil, "", nil, errors.New("extension is too large when extracted")
	}

	// manifest at the zip root, or inside exactly one top-level folder
	prefix := ""
	found := false
	for _, f := range zr.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if name == extManifestFileName {
			prefix, found = "", true
			break
		}
		parts := strings.Split(name, "/")
		if len(parts) == 2 && parts[1] == extManifestFileName && !found {
			prefix, found = parts[0]+"/", true
		}
	}
	if !found {
		return nil, "", nil, errors.New("manifest.json not found in zip")
	}

	var manifestFile *zip.File
	for _, f := range zr.File {
		if strings.ReplaceAll(f.Name, "\\", "/") == prefix+extManifestFileName {
			manifestFile = f
			break
		}
	}
	rc, err := manifestFile.Open()
	if err != nil {
		return nil, "", nil, errors.New("cannot read manifest.json")
	}
	defer rc.Close()
	raw, err := io.ReadAll(io.LimitReader(rc, 1<<20))
	if err != nil {
		return nil, "", nil, errors.New("cannot read manifest.json")
	}
	var mf ExtManifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, "", nil, errors.New("manifest.json is not valid JSON")
	}
	if err := normalizeManifest(&mf); err != nil {
		return nil, "", nil, err
	}

	// every referenced file must exist in the zip
	have := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		n := strings.ReplaceAll(f.Name, "\\", "/")
		if strings.HasPrefix(n, prefix) {
			have[strings.TrimPrefix(n, prefix)] = true
		}
	}
	for _, p := range append(append([]string{}, mf.JS...), mf.CSS...) {
		if !have[p] {
			return nil, "", nil, fmt.Errorf("file %q listed in manifest is missing", p)
		}
	}
	return &mf, prefix, zr, nil
}

func normalizeManifest(mf *ExtManifest) error {
	mf.ID = strings.TrimSpace(mf.ID)
	mf.Name = strings.TrimSpace(mf.Name)
	if !extIDPattern.MatchString(mf.ID) {
		return errors.New("manifest id must be lowercase letters, digits, '.', '-' or '_'")
	}
	if mf.Name == "" {
		return errors.New("manifest name is required")
	}
	if mf.Version == "" {
		mf.Version = "1.0.0"
	}
	switch mf.RunAt {
	case "":
		mf.RunAt = "document_idle"
	case "document_start", "document_end", "document_idle":
	default:
		return errors.New("run_at must be document_start, document_end or document_idle")
	}
	if len(mf.Matches) == 0 {
		mf.Matches = []string{"<all_urls>"}
	}
	for _, p := range append(append([]string{}, mf.Matches...), mf.ExcludeMatches...) {
		if _, err := compileMatchPattern(p); err != nil {
			return fmt.Errorf("bad match pattern %q", p)
		}
	}
	for _, p := range mf.HostPermissions {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(p)), "https://") {
			return fmt.Errorf("host permission %q must start with https://", p)
		}
		if _, err := compileMatchPattern(p); err != nil {
			return fmt.Errorf("bad host permission %q", p)
		}
	}
	clean := func(list []string) ([]string, error) {
		out := make([]string, 0, len(list))
		for _, p := range list {
			p = strings.ReplaceAll(p, "\\", "/")
			c := path.Clean(p)
			if c == "." || path.IsAbs(c) || c == ".." || strings.HasPrefix(c, "../") {
				return nil, fmt.Errorf("unsafe file path %q in manifest", p)
			}
			out = append(out, c)
		}
		return out, nil
	}
	var err error
	if mf.JS, err = clean(mf.JS); err != nil {
		return err
	}
	if mf.CSS, err = clean(mf.CSS); err != nil {
		return err
	}
	if len(mf.JS) == 0 && len(mf.CSS) == 0 {
		return errors.New("extension needs at least one js or css file")
	}
	return nil
}

// ---------- install / enable / disable / remove ----------

// InstallZip validates the zip and stores it as a package (disabled unless
// it was already enabled, in which case it is re-extracted = update).
func (m *ExtensionManager) InstallZip(data []byte) (*ExtInfo, error) {
	mf, _, _, err := readPackage(data)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.packagesDir(), 0o755); err != nil {
		return nil, err
	}
	tmp := m.packagePath(mf.ID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, m.packagePath(mf.ID)); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	if m.enabled[mf.ID] {
		if err := m.extractLocked(mf.ID); err == nil {
			if ext, err := loadExtFromDir(m.installedPath(mf.ID)); err == nil {
				m.loaded[mf.ID] = ext
			}
		}
	}
	return m.infoLocked(mf, len(data)), nil
}

// InstallURL downloads a zip (e.g. from the app store) and installs it.
func (m *ExtensionManager) InstallURL(rawURL string) (*ExtInfo, error) {
	rawURL = strings.TrimSpace(rawURL)
	low := strings.ToLower(rawURL)
	if !strings.HasPrefix(low, "https://") && !strings.HasPrefix(low, "http://") {
		return nil, errors.New("URL must start with http:// or https://")
	}
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("download failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, extMaxZipBytes+1))
	if err != nil {
		return nil, fmt.Errorf("download failed: %v", err)
	}
	return m.InstallZip(data)
}

// Enable extracts the zip and starts running the extension.
func (m *ExtensionManager) Enable(id string) error {
	if !extIDPattern.MatchString(id) {
		return errors.New("invalid extension id")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := os.Stat(m.packagePath(id)); err != nil {
		return errors.New("extension package not found")
	}
	if err := m.extractLocked(id); err != nil {
		return err
	}
	ext, err := loadExtFromDir(m.installedPath(id))
	if err != nil {
		return err
	}
	m.enabled[id] = true
	m.loaded[id] = ext
	m.saveStateLocked()
	return nil
}

// Disable stops the extension and deletes the extracted folder.
func (m *ExtensionManager) Disable(id string) error {
	if !extIDPattern.MatchString(id) {
		return errors.New("invalid extension id")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.enabled, id)
	delete(m.loaded, id)
	err := os.RemoveAll(m.installedPath(id))
	m.saveStateLocked()
	return err
}

// Remove disables the extension and deletes its zip package.
func (m *ExtensionManager) Remove(id string) error {
	if err := m.Disable(id); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := os.Remove(m.packagePath(id)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *ExtensionManager) extractLocked(id string) error {
	data, err := os.ReadFile(m.packagePath(id))
	if err != nil {
		return errors.New("extension package not found")
	}
	_, prefix, zr, err := readPackage(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.installedDir(), 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(m.installedDir(), ".extract-")
	if err != nil {
		return err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	for _, f := range zr.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rel := path.Clean(strings.TrimPrefix(name, prefix))
		if rel == "." || f.FileInfo().IsDir() {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			continue
		}
		// zip-slip protection
		if path.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, "../") || strings.Contains(rel, ":") {
			cleanup()
			return fmt.Errorf("unsafe path in package: %q", f.Name)
		}
		target := filepath.Join(tmp, filepath.FromSlash(rel))
		if !strings.HasPrefix(target, tmp+string(os.PathSeparator)) {
			cleanup()
			return fmt.Errorf("unsafe path in package: %q", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			cleanup()
			return err
		}
		rc, err := f.Open()
		if err != nil {
			cleanup()
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			cleanup()
			return err
		}
		_, copyErr := io.Copy(out, io.LimitReader(rc, extMaxFileBytes+1))
		rc.Close()
		out.Close()
		if copyErr != nil {
			cleanup()
			return copyErr
		}
	}

	final := m.installedPath(id)
	_ = os.RemoveAll(final)
	if err := os.Rename(tmp, final); err != nil {
		cleanup()
		return err
	}
	return nil
}

func loadExtFromDir(dir string) (*loadedExt, error) {
	raw, err := os.ReadFile(filepath.Join(dir, extManifestFileName))
	if err != nil {
		return nil, err
	}
	var mf ExtManifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, err
	}
	if err := normalizeManifest(&mf); err != nil {
		return nil, err
	}
	ext := &loadedExt{manifest: mf}

	var js strings.Builder
	for _, p := range mf.JS {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			return nil, err
		}
		js.Write(b)
		js.WriteString("\n;\n")
	}
	var css strings.Builder
	for _, p := range mf.CSS {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			return nil, err
		}
		css.Write(b)
		css.WriteString("\n")
	}
	ext.js, ext.css = js.String(), css.String()

	for _, p := range mf.Matches {
		re, err := compileMatchPattern(p)
		if err != nil {
			return nil, err
		}
		ext.include = append(ext.include, re)
	}
	for _, p := range mf.ExcludeMatches {
		re, err := compileMatchPattern(p)
		if err != nil {
			return nil, err
		}
		ext.exclude = append(ext.exclude, re)
	}
	for _, p := range mf.HostPermissions {
		re, err := compileMatchPattern(p)
		if err != nil {
			return nil, err
		}
		ext.hosts = append(ext.hosts, re)
	}
	return ext, nil
}

// ---------- listing ----------

func (m *ExtensionManager) infoLocked(mf *ExtManifest, size int) *ExtInfo {
	return &ExtInfo{
		ID: mf.ID, Name: mf.Name, Version: mf.Version, Description: mf.Description,
		Author: mf.Author, Icon: mf.Icon, Matches: mf.Matches,
		Enabled: m.enabled[mf.ID], SizeKB: (size + 1023) / 1024,
	}
}

// List returns every installed package (enabled or not), sorted by name.
func (m *ExtensionManager) List() []ExtInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := []ExtInfo{}
	entries, err := os.ReadDir(m.packagesDir())
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".zip") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(m.packagesDir(), e.Name()))
		if err != nil {
			continue
		}
		mf, _, _, err := readPackage(data)
		if err != nil {
			continue
		}
		out = append(out, *m.infoLocked(mf, len(data)))
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// ---------- URL matching ----------

// compileMatchPattern converts a Chrome-style match pattern into a regexp.
// Supported: "<all_urls>" (http/https/file) and "scheme://host/path" with *.
func compileMatchPattern(p string) (*regexp.Regexp, error) {
	p = strings.TrimSpace(p)
	if p == "<all_urls>" {
		return regexp.MustCompile(`(?i)^(https?|file)://.*$`), nil
	}
	i := strings.Index(p, "://")
	if i <= 0 {
		return nil, errors.New("missing scheme")
	}
	scheme, rest := p[:i], p[i+3:]
	var schemeRe string
	switch strings.ToLower(scheme) {
	case "*":
		schemeRe = `https?`
	case "http", "https", "file":
		schemeRe = regexp.QuoteMeta(strings.ToLower(scheme))
	default:
		return nil, errors.New("unsupported scheme")
	}
	host, pathPart := rest, "/*"
	if j := strings.Index(rest, "/"); j >= 0 {
		host, pathPart = rest[:j], rest[j:]
	}
	var hostRe string
	switch {
	case host == "*":
		hostRe = `[^/]*`
	case strings.HasPrefix(host, "*."):
		hostRe = `([^/]*\.)?` + regexp.QuoteMeta(host[2:])
	case host == "" && strings.ToLower(scheme) == "file":
		hostRe = ``
	case host == "":
		return nil, errors.New("missing host")
	default:
		hostRe = regexp.QuoteMeta(host)
	}
	pathRe := strings.ReplaceAll(regexp.QuoteMeta(pathPart), `\*`, `.*`)
	return regexp.Compile(`(?i)^` + schemeRe + `://` + hostRe + `(:\d+)?` + pathRe + `$`)
}

func (e *loadedExt) matches(pageURL string) bool {
	for _, re := range e.exclude {
		if re.MatchString(pageURL) {
			return false
		}
	}
	for _, re := range e.include {
		if re.MatchString(pageURL) {
			return true
		}
	}
	return false
}

// ---------- injection payloads ----------

const extRunTemplate = `(function(){
var ID=__ID__,U=__URL__,RUNAT=__RUNAT__,CSS=__CSS__;
try{
  if(location.href!==U){return;}
  var ran=(window.__sparrowExtRan=window.__sparrowExtRan||{});
  if(ran[ID]){return;}
  ran[ID]=true;
}catch(e){return;}
function inject(){
  if(CSS){
    try{
      var sh=new CSSStyleSheet();sh.replaceSync(CSS);
      document.adoptedStyleSheets=document.adoptedStyleSheets.concat([sh]);
    }catch(e){
      try{
        var st=document.createElement('style');
        st.setAttribute('data-sparrow-ext',ID);
        st.textContent=CSS;
        (document.head||document.documentElement).appendChild(st);
      }catch(_){}
    }
  }
  var pend=(window.__sparrowExtPending=window.__sparrowExtPending||{});
  if(!window.__sparrowExtDone){
    window.__sparrowExtDone=function(rid,raw){
      var p=pend[rid];if(!p){return;}delete pend[rid];
      try{var r=JSON.parse(raw);
        if(r.ok){p.res({status:r.status,text:r.body});}else{p.rej(new Error(r.error||'request failed'));}
      }catch(e){p.rej(e);}
    };
  }
  function sparrowFetch(u){
    return new Promise(function(res,rej){
      if(typeof window.extFetchStart!=='function'){rej(new Error('fetch is not available'));return;}
      var rid=ID+':'+Math.random().toString(36).slice(2);
      pend[rid]={res:res,rej:rej};
      setTimeout(function(){if(pend[rid]){delete pend[rid];rej(new Error('timeout'));}},20000);
      try{window.extFetchStart(ID,String(u),rid);}catch(e){delete pend[rid];rej(e);}
    });
  }
  try{
    (function(sparrowExt){
__JS__
    })({id:ID,version:__VERSION__,fetch:sparrowFetch});
  }catch(err){
    try{console.error('[Sparrow extension '+ID+']',err);}catch(_){}
  }
}
var rs=document.readyState;
if(RUNAT==='document_start'||rs!=='loading'){inject();}
else{document.addEventListener('DOMContentLoaded',inject,{once:true});}
})();`

func jsString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// PayloadsFor returns one self-contained script per enabled extension that
// matches pageURL. Each script is evaluated separately so one broken
// extension can never break another.
func (m *ExtensionManager) PayloadsFor(pageURL string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]string, 0, len(m.loaded))
	for id := range m.loaded {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var out []string
	for _, id := range ids {
		ext := m.loaded[id]
		if ext == nil || !ext.matches(pageURL) {
			continue
		}
		r := strings.NewReplacer(
			"__ID__", jsString(ext.manifest.ID),
			"__URL__", jsString(pageURL),
			"__RUNAT__", jsString(ext.manifest.RunAt),
			"__CSS__", jsString(ext.css),
			"__VERSION__", jsString(ext.manifest.Version),
		)
		code := r.Replace(extRunTemplate)
		// the extension's own code goes in last so it is never rewritten
		code = strings.Replace(code, "__JS__", ext.js, 1)
		out = append(out, code)
	}
	return out
}

// ---------- restricted network access for extensions ----------

var extHTTPClient = &http.Client{Timeout: 15 * time.Second}

const extFetchMaxBytes = 2 << 20

// Fetch performs an https GET for an enabled extension. The URL must match
// one of the extension's host_permissions.
func (m *ExtensionManager) Fetch(id, rawURL string) (int, string, error) {
	m.mu.Lock()
	ext := m.loaded[id]
	m.mu.Unlock()
	if ext == nil {
		return 0, "", errors.New("extension is not enabled")
	}
	low := strings.ToLower(rawURL)
	if !strings.HasPrefix(low, "https://") {
		return 0, "", errors.New("only https URLs are allowed")
	}
	allowed := false
	for _, re := range ext.hosts {
		if re.MatchString(rawURL) {
			allowed = true
			break
		}
	}
	if !allowed {
		return 0, "", errors.New("host is not in the extension's host_permissions")
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Sparrow")
	resp, err := extHTTPClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, extFetchMaxBytes))
	if err != nil {
		return resp.StatusCode, "", err
	}
	return resp.StatusCode, string(body), nil
}

func extFetchJSON(id, rawURL string) string {
	status, body, err := extensions.Fetch(id, rawURL)
	if err != nil {
		b, _ := json.Marshal(map[string]interface{}{"ok": false, "error": err.Error()})
		return string(b)
	}
	b, _ := json.Marshal(map[string]interface{}{"ok": true, "status": status, "body": body})
	return string(b)
}

// ---------- JSON helpers used by the bindings ----------

func extResult(ok bool, msg string, data interface{}) string {
	b, _ := json.Marshal(map[string]interface{}{"ok": ok, "error": msg, "data": data})
	return string(b)
}

func extCheckToken(token string) bool {
	return token != "" && token == extUIToken
}

func extListJSON(token string) string {
	if !extCheckToken(token) {
		return extResult(false, "not allowed", nil)
	}
	return extResult(true, "", extensions.List())
}

func extSetEnabledJSON(token, id string, on bool) string {
	if !extCheckToken(token) {
		return extResult(false, "not allowed", nil)
	}
	var err error
	if on {
		err = extensions.Enable(id)
	} else {
		err = extensions.Disable(id)
	}
	if err != nil {
		return extResult(false, err.Error(), nil)
	}
	return extResult(true, "", nil)
}

func extInstallB64JSON(token, b64 string) string {
	if !extCheckToken(token) {
		return extResult(false, "not allowed", nil)
	}
	if i := strings.Index(b64, ","); i >= 0 && strings.HasPrefix(b64, "data:") {
		b64 = b64[i+1:]
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return extResult(false, "could not read the file", nil)
	}
	info, err := extensions.InstallZip(data)
	if err != nil {
		return extResult(false, err.Error(), nil)
	}
	return extResult(true, "", info)
}

func extInstallURLJSON(token, rawURL string) string {
	if !extCheckToken(token) {
		return extResult(false, "not allowed", nil)
	}
	info, err := extensions.InstallURL(rawURL)
	if err != nil {
		return extResult(false, err.Error(), nil)
	}
	return extResult(true, "", info)
}

func extRemoveJSON(token, id string) string {
	if !extCheckToken(token) {
		return extResult(false, "not allowed", nil)
	}
	if err := extensions.Remove(id); err != nil {
		return extResult(false, err.Error(), nil)
	}
	return extResult(true, "", nil)
}

// extensionLoaderJS runs on every page (registered with w.Init). In the top
// frame of http/https/file pages it asks Go for the matching extensions.
// Go answers by evaluating the extension code in the page (this is not
// affected by the site's Content-Security-Policy).
const extensionLoaderJS = `(function(){
  try{
    if(window.top!==window){return;}
    var u=location.href;
    if(!/^(https?|file):/i.test(u)){return;}
    if(window.__sparrowExtAsked===u){return;}
    window.__sparrowExtAsked=u;
    var tries=0;
    (function ask(){
      if(typeof window.extRequest==='function'){
        try{window.extRequest(u);}catch(e){}
        return;
      }
      if(++tries<100){setTimeout(ask,25);}
    })();
  }catch(e){}
})();`
