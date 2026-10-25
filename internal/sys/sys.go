// Package sys makes the one-click system changes: a firewall rule for Van
// Link (after the operating system's own permission prompt), keeping the
// computer awake during a broadcast, and undoing it all.
package sys

// RuleName is the firewall rule Van Link adds and removes.
const RuleName = "Van Link"

// SetupResult tells the screen what happened.
type SetupResult struct {
	Done    bool   `json:"done"`
	Message string `json:"message"`
}
