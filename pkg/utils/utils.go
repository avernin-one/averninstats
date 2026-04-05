package utils

import (
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
	"gopkg.in/yaml.v3"
)

// SaveYAMLFile encodes data as YAML and writes it to filePath, creating
// intermediate directories as needed.
func SaveYAMLFile(filePath string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		return fmt.Errorf("create directory %q: %w", filepath.Dir(filePath), err)
	}

	out, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer out.Close()

	enc := yaml.NewEncoder(out)
	enc.SetIndent(2)
	defer enc.Close()

	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode YAML to %q: %w", filePath, err)
	}

	log.Debug().Str("filepath", filePath).Msg("saved file")

	return nil
}

// SaveJSONFile encodes data as indented JSON and writes it to filePath,
// creating intermediate directories as needed.
func SaveJSONFile(filePath string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		return fmt.Errorf("create directory %q: %w", filepath.Dir(filePath), err)
	}

	out, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer out.Close()

	var jsonData []byte
	if config.Get().Minify {
		jsonData, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal JSON data for %q: %w", filePath, err)
		}
	} else {
		jsonData, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal indent JSON to %q: %w", filePath, err)
		}
	}

	_, err = out.Write(jsonData)
	if err != nil {
		return fmt.Errorf("write JSON data to %q: %w", filePath, err)
	}

	log.Debug().Str("filepath", filePath).Msg("saved file")

	return nil
}

func ReadJSONFile(filePath string, out interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file %q: %w", filePath, err)
	}

	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal JSON from %q: %w", filePath, err)
	}

	return nil
}

// Returns true if the file at filePath exists.
// When notExistentIfEmpty is true it also returns false for zero-byte files
// even tho they exist.
func FileExists(filePath string, notExistentIfEmpty bool) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		return false
	}

	if notExistentIfEmpty && info.Size() == 0 {
		return false
	}

	return true
}

// NewHttpRequest performs a GET request to url and returns the response body.
func NewHttpRequest(url string) ([]byte, error) {
	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %q: %w", url, err)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %q: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %q returned status %d", url, res.StatusCode)
	}

	return io.ReadAll(res.Body)
}

// AddRandomTime adds a random number of hours in [0, extraHours) to currentTime.
func AddRandomTime(currentTime time.Time, extraHours int) time.Time {
	if extraHours == 0 {
		return currentTime
	}

	return currentTime.Add(time.Duration(rand.Intn(extraHours)) * time.Hour) //nolint:gosec // rand.Intn is sufficient for this use case
}

// https://stackoverflow.com/a/51997907
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
