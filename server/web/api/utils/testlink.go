package utils

import (
	"fmt"
	"net/url"
)

// TestLink checks the canonical link that the caller will execute. Do not
// unescape again here: validation and parsing must see identical URL components.
func TestLink(link string, auth bool) error {
	ur, err := url.Parse(link)
	if err != nil {
		return err
	}

	if ur.Scheme == "magnet" || ur.Scheme == "torrs" || ur.Scheme == "" {
		return nil
	}

	if !auth {
		return fmt.Errorf("auth required")
	}

	return nil
}
