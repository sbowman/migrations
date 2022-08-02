package cmd

import (
	"fmt"
	"os"
)

// Execute configures the command structures for Concierge.
func Execute() {
	if err := root.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to migrate: %s", err)
	}
}
