// Package pwned checks passwords against Have I Been Pwned's Pwned Passwords,
// a collection of passwords seen in data breaches.
package pwned

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultURL is the public Pwned Passwords API.
const DefaultURL = "https://api.pwnedpasswords.com"

// Client queries the Pwned Passwords range API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client for the API at baseURL. Each request gives up
// after timeout.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

// Count returns how many times the password has been seen in breaches, or 0
// if never. Only the first 5 characters of the password's SHA-1 hash are
// sent, and the response is padded with dummy entries, so the API learns
// neither the password nor which one was checked.
func (c *Client) Count(ctx context.Context, password string) (int, error) {
	sum := sha1.Sum([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix, suffix := hash[:5], hash[5:]

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/range/"+prefix, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Add-Padding", "true")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("pwned passwords: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("pwned passwords: unexpected status %d", resp.StatusCode)
	}

	// Each line is "<35 more hash characters>:<count>". Padding lines have a
	// count of 0, so a match on one would correctly count as not breached.
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line, count, ok := strings.Cut(strings.TrimSpace(scanner.Text()), ":")
		if ok && strings.EqualFold(line, suffix) {
			n, err := strconv.Atoi(count)
			if err != nil {
				return 0, fmt.Errorf("pwned passwords: bad count %q: %w", count, err)
			}
			return n, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("pwned passwords: read response: %w", err)
	}
	return 0, nil
}
