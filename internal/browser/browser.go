package browser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Browser manages a browser instance for automation.
type Browser struct {
	mu        sync.Mutex
	browser   *rod.Browser
	running   bool
	headless  bool
	connected bool // true when attached to external Chrome (don't close on Stop)
	dataDir   string

	// refs holds the latest snapshot ref map (ref number → RefEntry).
	refs map[int]RefEntry
}

var (
	instance *Browser
	once     sync.Once
)

// Instance returns the singleton browser manager.
func Instance() *Browser {
	once.Do(func() {
		home, _ := os.UserHomeDir()
		instance = &Browser{
			headless: false,
			dataDir:  filepath.Join(home, ".lingti-bot", "browser"),
			refs:     make(map[int]RefEntry),
		}
	})
	return instance
}

// StartOptions configures browser launch.
type StartOptions struct {
	Headless       bool
	ExecutablePath string
	URL            string
	ConnectURL     string // CDP address to connect to existing Chrome (e.g. "127.0.0.1:9222")
}

// Start launches a new browser instance or connects to an existing one.
func (b *Browser) Start(opts StartOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return fmt.Errorf("browser already running")
	}

	// Connect to existing Chrome via CDP
	if opts.ConnectURL != "" {
		return b.connectLocked(opts.ConnectURL, opts.URL)
	}

	b.headless = opts.Headless

	if err := os.MkdirAll(b.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	l := launcher.New().
		UserDataDir(b.dataDir).
		Headless(opts.Headless)

	// Use specified executable, or auto-detect Chrome
	bin := opts.ExecutablePath
	if bin == "" {
		bin = detectChrome()
	}
	if bin != "" {
		l = l.Bin(bin)
	}

	controlURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	brow := rod.New().ControlURL(controlURL)
	if err := brow.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	b.browser = brow
	b.running = true
	b.connected = false
	b.refs = make(map[int]RefEntry)

	if opts.URL != "" {
		page, err := brow.Page(proto.TargetCreateTarget{URL: opts.URL})
		if err != nil {
			return fmt.Errorf("failed to open initial page: %w", err)
		}
		page.MustWaitStable()
	}

	return nil
}

// connectLocked connects to an existing Chrome at the given CDP address.
// Must be called with b.mu held.
func (b *Browser) connectLocked(addr string, initialURL string) error {
	controlURL, err := launcher.ResolveURL(addr)
	if err != nil {
		return fmt.Errorf("failed to resolve CDP address %s (is Chrome running with --remote-debugging-port?): %w", addr, err)
	}

	brow := rod.New().ControlURL(controlURL)
	if err := brow.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser at %s: %w", addr, err)
	}

	b.browser = brow
	b.running = true
	b.connected = true
	b.refs = make(map[int]RefEntry)

	if initialURL != "" {
		page, err := brow.Page(proto.TargetCreateTarget{URL: initialURL})
		if err != nil {
			return fmt.Errorf("failed to open initial page: %w", err)
		}
		page.MustWaitStable()
	}

	return nil
}

// Stop closes the browser (or just disconnects if attached to external Chrome).
func (b *Browser) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return fmt.Errorf("browser not running")
	}

	if !b.connected {
		if err := b.browser.Close(); err != nil {
			return fmt.Errorf("failed to close browser: %w", err)
		}
	}
	// When connected to external Chrome, just drop the reference — don't close it.

	b.browser = nil
	b.running = false
	b.connected = false
	b.refs = make(map[int]RefEntry)
	return nil
}

// EnsureRunning starts the browser if not already running.
// Tries to connect to existing Chrome on port 9222 first, then launches a new one.
func (b *Browser) EnsureRunning() error {
	b.mu.Lock()
	running := b.running
	b.mu.Unlock()

	if running {
		return nil
	}

	// Try connecting to existing Chrome with debugging port
	if _, err := launcher.ResolveURL("127.0.0.1:9222"); err == nil {
		return b.Start(StartOptions{ConnectURL: "127.0.0.1:9222"})
	}

	return b.Start(StartOptions{
		Headless:       false,
		ExecutablePath: detectChrome(),
	})
}

// IsRunning returns whether the browser is active.
func (b *Browser) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

// IsConnected returns whether the browser is attached to an external Chrome.
func (b *Browser) IsConnected() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.connected
}

// Rod returns the underlying rod browser. Caller must check IsRunning first.
func (b *Browser) Rod() *rod.Browser {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.browser
}

// ActivePage returns the most recently used page or the first available page.
func (b *Browser) ActivePage() (*rod.Page, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil, fmt.Errorf("browser not running")
	}

	pages, err := b.browser.Pages()
	if err != nil {
		return nil, fmt.Errorf("failed to get pages: %w", err)
	}
	if len(pages) == 0 {
		// Create a blank page if none exist
		page, err := b.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
		if err != nil {
			return nil, fmt.Errorf("failed to create page: %w", err)
		}
		return page, nil
	}
	return pages.First(), nil
}

// SetRefs stores the ref map from a snapshot.
func (b *Browser) SetRefs(refs map[int]RefEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refs = refs
}

// GetRef returns a ref entry by number.
func (b *Browser) GetRef(ref int) (RefEntry, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	entry, ok := b.refs[ref]
	return entry, ok
}

// StatusInfo holds browser status details.
type StatusInfo struct {
	Running   bool   `json:"running"`
	Headless  bool   `json:"headless"`
	Connected bool   `json:"connected"` // attached to external Chrome (vs launched)
	Pages     int    `json:"pages"`
	ActiveURL string `json:"active_url"`
}

// ExecuteJS runs JavaScript on the active page and returns the result as a string.
func ExecuteJS(page *rod.Page, script string) (string, error) {
	result, err := page.Eval(script)
	if err != nil {
		return "", fmt.Errorf("JS execution failed: %w", err)
	}
	return result.Value.String(), nil
}

// detectChrome returns the path to a local Chrome installation, or empty string if not found.
func detectChrome() string {
	switch runtime.GOOS {
	case "darwin":
		candidates := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	case "linux":
		for _, name := range []string{"google-chrome", "google-chrome-stable"} {
			if p, err := exec.LookPath(name); err == nil {
				return p
			}
		}
	}
	return ""
}

// Status returns current browser state.
func (b *Browser) Status() StatusInfo {
	b.mu.Lock()
	defer b.mu.Unlock()

	info := StatusInfo{
		Running:   b.running,
		Headless:  b.headless,
		Connected: b.connected,
	}

	if !b.running {
		return info
	}

	pages, err := b.browser.Pages()
	if err == nil {
		info.Pages = len(pages)
		if len(pages) > 0 {
			pageInfo, err := pages.First().Info()
			if err == nil {
				info.ActiveURL = pageInfo.URL
			}
		}
	}
	return info
}
