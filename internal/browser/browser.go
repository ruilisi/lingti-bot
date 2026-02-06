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
	mu       sync.Mutex
	browser  *rod.Browser
	running  bool
	headless bool
	dataDir  string

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
}

// Start launches a new browser instance.
func (b *Browser) Start(opts StartOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return fmt.Errorf("browser already running")
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

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	b.browser = browser
	b.running = true
	b.refs = make(map[int]RefEntry)

	if opts.URL != "" {
		page, err := browser.Page(proto.TargetCreateTarget{URL: opts.URL})
		if err != nil {
			return fmt.Errorf("failed to open initial page: %w", err)
		}
		page.MustWaitStable()
	}

	return nil
}

// Stop closes the browser.
func (b *Browser) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return fmt.Errorf("browser not running")
	}

	if err := b.browser.Close(); err != nil {
		return fmt.Errorf("failed to close browser: %w", err)
	}

	b.browser = nil
	b.running = false
	b.refs = make(map[int]RefEntry)
	return nil
}

// EnsureRunning starts the browser if not already running.
// Defaults to headed mode with local Chrome if available.
func (b *Browser) EnsureRunning() error {
	b.mu.Lock()
	running := b.running
	b.mu.Unlock()

	if running {
		return nil
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
		Running:  b.running,
		Headless: b.headless,
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
