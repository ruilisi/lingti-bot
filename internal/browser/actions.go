package browser

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// Click clicks the element identified by the given ref number.
// It scrolls the element into view, waits for it to be interactable, then clicks.
func Click(page *rod.Page, b *Browser, ref int) error {
	el, err := resolveRef(page, b, ref)
	if err != nil {
		return err
	}

	// Scroll into view so the element is visible
	if err := el.ScrollIntoView(); err != nil {
		// Not fatal — element might already be in view
		_ = err
	}

	// Wait for element to be interactable (visible + not covered)
	if _, err := el.Interactable(); err != nil {
		// Try a short wait and retry once
		time.Sleep(500 * time.Millisecond)
		if _, err := el.Interactable(); err != nil {
			return fmt.Errorf("element [%d] not interactable: %w", ref, err)
		}
	}

	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click failed: %w", err)
	}

	// Wait briefly for page to settle after click (animations, AJAX, etc.)
	_ = page.WaitStable(300 * time.Millisecond)

	return nil
}

// Type inputs text into the element identified by the given ref number.
// It clicks the element first to ensure focus, then types.
func Type(page *rod.Page, b *Browser, ref int, text string, submit bool) error {
	el, err := resolveRef(page, b, ref)
	if err != nil {
		return err
	}

	// Scroll into view
	if err := el.ScrollIntoView(); err != nil {
		_ = err
	}

	// Click to focus the element first — critical for search boxes, inputs, etc.
	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		// Try Focus as fallback
		if err := el.Focus(); err != nil {
			return fmt.Errorf("failed to focus element [%d]: %w", ref, err)
		}
	}

	// Small delay to let focus animations/handlers run
	time.Sleep(200 * time.Millisecond)

	// Clear existing content
	if err := el.SelectAllText(); err != nil {
		// Element might not have text, ignore
		_ = err
	}

	// Input text
	if err := el.Input(text); err != nil {
		return fmt.Errorf("failed to type text: %w", err)
	}

	if submit {
		// Small delay before submit to let input handlers process
		time.Sleep(100 * time.Millisecond)
		if err := el.Type(input.Enter); err != nil {
			return fmt.Errorf("failed to press Enter: %w", err)
		}
		// Wait for page to settle after submit (navigation/AJAX)
		_ = page.WaitStable(500 * time.Millisecond)
	}

	return nil
}

// Press sends a keyboard key press to the page.
func Press(page *rod.Page, key string) error {
	k, ok := keyMap[key]
	if !ok {
		return fmt.Errorf("unknown key: %q (supported: Enter, Tab, Escape, Backspace, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Space, Delete, Home, End, PageUp, PageDown)", key)
	}
	if err := page.Keyboard.Type(k); err != nil {
		return err
	}
	// Wait for page to settle after key press
	_ = page.WaitStable(300 * time.Millisecond)
	return nil
}

// Hover moves the mouse over the element identified by the given ref number.
func Hover(page *rod.Page, b *Browser, ref int) error {
	el, err := resolveRef(page, b, ref)
	if err != nil {
		return err
	}
	if err := el.ScrollIntoView(); err != nil {
		_ = err
	}
	return el.Hover()
}

// ClickAll clicks every element matching the CSS selector with a delay between each.
// Returns the number of elements successfully clicked.
func ClickAll(page *rod.Page, selector string, delay time.Duration) (int, error) {
	elements, err := page.Elements(selector)
	if err != nil {
		return 0, fmt.Errorf("failed to find elements matching %q: %w", selector, err)
	}

	clicked := 0
	for _, el := range elements {
		_ = el.ScrollIntoView()
		time.Sleep(200 * time.Millisecond)

		if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
			// Skip unclickable elements (hidden, covered, etc.)
			continue
		}
		clicked++

		if delay > 0 {
			time.Sleep(delay)
		}
	}

	return clicked, nil
}

// resolveRef looks up a ref number in the browser's ref map and returns the corresponding element.
func resolveRef(page *rod.Page, b *Browser, ref int) (*rod.Element, error) {
	entry, ok := b.GetRef(ref)
	if !ok {
		return nil, fmt.Errorf("ref %d not found in snapshot (run browser_snapshot first, or page may have changed)", ref)
	}

	if entry.BackendDOMNodeID == 0 {
		return nil, fmt.Errorf("ref %d has no backend DOM node (element may be virtual)", ref)
	}

	return resolveBackendNode(page, entry.BackendDOMNodeID)
}

// resolveBackendNode converts a BackendDOMNodeID to a rod Element.
func resolveBackendNode(page *rod.Page, backendNodeID proto.DOMBackendNodeID) (*rod.Element, error) {
	result, err := proto.DOMResolveNode{
		BackendNodeID: backendNodeID,
	}.Call(page)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve node (element may have been removed from page): %w", err)
	}

	if result.Object.ObjectID == "" {
		return nil, fmt.Errorf("resolved node has no object ID")
	}

	el, err := page.ElementFromObject(result.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to create element from object: %w", err)
	}
	return el, nil
}

// keyMap maps human-readable key names to rod input keys.
var keyMap = map[string]input.Key{
	"Enter":      input.Enter,
	"Tab":        input.Tab,
	"Escape":     input.Escape,
	"Backspace":  input.Backspace,
	"Delete":     input.Delete,
	"ArrowUp":    input.ArrowUp,
	"ArrowDown":  input.ArrowDown,
	"ArrowLeft":  input.ArrowLeft,
	"ArrowRight": input.ArrowRight,
	"Space":      input.Space,
	"Home":       input.Home,
	"End":        input.End,
	"PageUp":     input.PageUp,
	"PageDown":   input.PageDown,
}
