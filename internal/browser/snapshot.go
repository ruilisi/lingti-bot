package browser

import (
	"fmt"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// RefEntry maps a snapshot ref to a backend DOM node.
type RefEntry struct {
	BackendDOMNodeID proto.DOMBackendNodeID `json:"backend_dom_node_id"`
	Role             string                 `json:"role"`
	Name             string                 `json:"name"`
}

// interactiveRoles are roles that represent user-interactive elements.
var interactiveRoles = map[string]bool{
	"button":           true,
	"link":             true,
	"textbox":          true,
	"searchbox":        true,
	"combobox":         true,
	"checkbox":         true,
	"radio":            true,
	"switch":           true,
	"slider":           true,
	"spinbutton":       true,
	"tab":              true,
	"menuitem":         true,
	"menuitemcheckbox": true,
	"menuitemradio":    true,
	"option":           true,
	"treeitem":         true,
	"gridcell":         true,
	"row":              true,
	"columnheader":     true,
	"rowheader":        true,
}

// visibleRoles are non-interactive but informational roles worth showing.
var visibleRoles = map[string]bool{
	"heading":       true,
	"img":           true,
	"table":         true,
	"list":          true,
	"listitem":      true,
	"navigation":    true,
	"main":          true,
	"dialog":        true,
	"alert":         true,
	"status":        true,
	"banner":        true,
	"complementary": true,
	"contentinfo":   true,
	"form":          true,
	"region":        true,
	"article":       true,
	"cell":          true,
}

// Snapshot captures the accessibility tree and returns formatted text with refs.
func Snapshot(page *rod.Page) (string, map[int]RefEntry, error) {
	// Get the full accessibility tree
	nodes, err := proto.AccessibilityGetFullAXTree{}.Call(page)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}

	if len(nodes.Nodes) == 0 {
		return "(empty page)", nil, nil
	}

	// Build parent→children map
	childMap := make(map[proto.AccessibilityAXNodeID][]proto.AccessibilityAXNodeID)
	nodeMap := make(map[proto.AccessibilityAXNodeID]*proto.AccessibilityAXNode)
	var rootID proto.AccessibilityAXNodeID

	for i, node := range nodes.Nodes {
		nodeMap[node.NodeID] = node
		if i == 0 {
			rootID = node.NodeID
		}
		for _, childID := range node.ChildIDs {
			childMap[node.NodeID] = append(childMap[node.NodeID], childID)
		}
	}

	refs := make(map[int]RefEntry)
	refCounter := 1
	var sb strings.Builder

	// Walk the tree depth-first
	var walk func(id proto.AccessibilityAXNodeID, depth int)
	walk = func(id proto.AccessibilityAXNodeID, depth int) {
		node, ok := nodeMap[id]
		if !ok {
			return
		}

		// Skip ignored nodes
		if node.Ignored {
			// Still walk children of ignored nodes (they may have visible descendants)
			for _, childID := range childMap[id] {
				walk(childID, depth)
			}
			return
		}

		role := axValueString(node.Role)
		name := axValueString(node.Name)

		// Skip generic/root nodes without useful info
		if role == "none" || role == "generic" || role == "RootWebArea" || role == "WebArea" {
			for _, childID := range childMap[id] {
				walk(childID, depth)
			}
			return
		}

		// Skip unnamed non-interactive structural nodes
		if name == "" && !interactiveRoles[role] && !visibleRoles[role] {
			for _, childID := range childMap[id] {
				walk(childID, depth)
			}
			return
		}

		// Determine if this node gets a ref (interactive or named visible)
		isInteractive := interactiveRoles[role]
		isVisible := visibleRoles[role] && name != ""

		indent := strings.Repeat("  ", depth)

		if isInteractive || isVisible {
			ref := refCounter
			refCounter++

			if node.BackendDOMNodeID != 0 {
				refs[ref] = RefEntry{
					BackendDOMNodeID: node.BackendDOMNodeID,
					Role:             role,
					Name:             name,
				}
			}

			if name != "" {
				fmt.Fprintf(&sb, "%s[%d] %s %q\n", indent, ref, role, name)
			} else {
				fmt.Fprintf(&sb, "%s[%d] %s\n", indent, ref, role)
			}
		}

		// Walk children
		for _, childID := range childMap[id] {
			walk(childID, depth+1)
		}
	}

	walk(rootID, 0)

	result := sb.String()
	if result == "" {
		result = "(no interactive elements found)"
	}

	return result, refs, nil
}

// axValueString extracts the string value from an accessibility Value.
func axValueString(v *proto.AccessibilityAXValue) string {
	if v == nil {
		return ""
	}
	return v.Value.Str()
}
