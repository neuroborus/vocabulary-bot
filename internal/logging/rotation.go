package logging

import (
	"fmt"
	"os"
)

// TruncateFile clears the active log file after successful delivery.
func TruncateFile(path string) error {
	if err := os.Truncate(path, 0); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("truncate log file: %w", err)
	}

	return nil
}
