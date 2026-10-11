package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

var (
	// Different sources mention that the Mojang API has a rate limit of
	// 1000 requests per 10 minutes or 200 per 2 minutes. We make 95 per minute
	// just to be safe.
	limiter = rate.NewLimiter(rate.Every(time.Minute/95), 5)

	httpClient = &http.Client{Timeout: 10 * time.Second}
)

// Upper bound for response bodies; the largest files we fetch (asset index,
// language files) are well below this.
const maxResponseSize = 32 << 20

// Encodes data as minified or indented JSON and writes it to,
// outFile creating intermediate directories as needed.
func SaveJSONFile(outFile string, data any) error {
	if err := os.MkdirAll(filepath.Dir(outFile), 0o750); err != nil {
		return fmt.Errorf("create directory %q: %w", filepath.Dir(outFile), err)
	}

	var (
		jsonData []byte
		err      error
	)
	if config.Get().Minify {
		jsonData, err = json.Marshal(data)
	} else {
		jsonData, err = json.MarshalIndent(data, "", "  ")
	}
	if err != nil {
		return fmt.Errorf("marshal JSON data for %q: %w", outFile, err)
	}

	if err := os.WriteFile(outFile, jsonData, 0o600); err != nil {
		return fmt.Errorf("write JSON data to %q: %w", outFile, err)
	}

	log.Debug().Str("file", outFile).Msg("saved file")

	return nil
}

func ReadJSONFile(inFile string, out any) error {
	data, err := os.ReadFile(inFile)
	if err != nil {
		return fmt.Errorf("read file %q: %w", inFile, err)
	}

	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal JSON from %q: %w", inFile, err)
	}

	return nil
}

// Returns true if the file at filePath exists.
// When notExistentIfEmpty is true it also returns false for zero-byte files
// even tho they exist.
func FileExists(filePath string, notExistentIfEmpty bool) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	if notExistentIfEmpty && info.Size() == 0 {
		return false
	}

	return true
}

// Perform limited GET request to url and returns the response body.
func NewHttpRequest(url string) ([]byte, error) {
	if err := limiter.Wait(context.Background()); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %q: %w", url, err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %q: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %q returned status %d", url, res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read body of %q: %w", url, err)
	}

	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("GET %q: response exceeds %d bytes", url, maxResponseSize)
	}

	return body, nil
}

// Adds a random number of hours between 0 and `addHours` to the current time.
func AddRandomTime(current time.Time, addHours int) time.Time {
	if addHours == 0 {
		return current
	}

	return current.Add(time.Duration(rand.Intn(addHours)) * time.Hour) //nolint:gosec // rand.Intn is sufficient for this use case
}

// https://stackoverflow.com/a/51997907
// Compare 2 strings case-insensitive.
func LessLower(sa, sb string) bool {
	for {
		rb, nb := utf8.DecodeRuneInString(sb)
		if nb == 0 {
			// The number of runes in sa is greater than or
			// equal to the number of runes in sb. It follows
			// that sa is not less than sb.
			return false
		}

		ra, na := utf8.DecodeRuneInString(sa)
		if na == 0 {
			// The number of runes in sa is less than the
			// number of runes in sb. It follows that sa
			// is less than sb.
			return true
		}

		rb = unicode.ToLower(rb)
		ra = unicode.ToLower(ra)

		if ra != rb {
			return ra < rb
		}

		// Trim rune from the beginning of each string.
		sa = sa[na:]
		sb = sb[nb:]
	}
}
