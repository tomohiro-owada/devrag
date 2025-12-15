package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	GitHubAPIURL     = "https://api.github.com/repos/tomohiro-owada/devrag/releases/latest"
	CheckIntervalHrs = 24
	CacheFileName    = ".devrag_update_check"
)

// semverRegex matches semantic versioning pattern (e.g., "1.2.3", "v1.2.3")
var semverRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)`)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type UpdateCache struct {
	LastCheck       time.Time `json:"last_check"`
	LatestVersion   string    `json:"latest_version"`
	NotifiedVersion string    `json:"notified_version"`
}

// UpdateInfo contains information about available update
type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	URL            string `json:"url"`
}

// GetUpdateInfo checks for updates and returns info (for MCP response)
// Returns nil if no update available or check was done recently
func GetUpdateInfo(currentVersion string, cacheDir string) *UpdateInfo {
	debug := os.Getenv("DEVRAG_DEBUG") != ""

	cache, err := loadCache(cacheDir)
	if err != nil && debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Failed to load update cache: %v\n", err)
	}

	// Check if we should notify (24h since last notification)
	if cache.NotifiedVersion != "" && time.Since(cache.LastCheck) < time.Duration(CheckIntervalHrs)*time.Hour {
		return nil // Already notified recently
	}

	// Fetch latest release
	release, err := fetchLatestRelease()
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to fetch latest release: %v\n", err)
		}
		return nil
	}

	latestVersion, err := normalizeVersion(release.TagName)
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Invalid version tag %q: %v\n", release.TagName, err)
		}
		return nil
	}

	newer, err := isNewerVersion(latestVersion, currentVersion)
	if err != nil || !newer {
		return nil
	}

	// Update cache to mark as notified
	cache.LastCheck = time.Now()
	cache.LatestVersion = latestVersion
	cache.NotifiedVersion = latestVersion
	if err := saveCache(cacheDir, cache); err != nil && debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Failed to save update cache: %v\n", err)
	}

	return &UpdateInfo{
		Available:      true,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		URL:            "https://github.com/tomohiro-owada/devrag/releases/latest",
	}
}

// CheckForUpdate checks GitHub for a newer version
// Errors are logged to stderr with debug context when DEVRAG_DEBUG is set
func CheckForUpdate(currentVersion string, cacheDir string) {
	debug := os.Getenv("DEVRAG_DEBUG") != ""

	cache, err := loadCache(cacheDir)
	if err != nil && debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Failed to load update cache: %v\n", err)
	}

	// Skip if checked within the last 24 hours
	if time.Since(cache.LastCheck) < time.Duration(CheckIntervalHrs)*time.Hour {
		// Still notify if there's a newer version we haven't notified about
		if cache.LatestVersion != "" && cache.NotifiedVersion != cache.LatestVersion {
			newer, err := isNewerVersion(cache.LatestVersion, currentVersion)
			if err != nil && debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Version comparison failed: %v\n", err)
			}
			if newer {
				printUpdateNotice(currentVersion, cache.LatestVersion)
				cache.NotifiedVersion = cache.LatestVersion
				if err := saveCache(cacheDir, cache); err != nil && debug {
					fmt.Fprintf(os.Stderr, "[DEBUG] Failed to save update cache: %v\n", err)
				}
			}
		}
		return
	}

	// Fetch latest release from GitHub
	release, err := fetchLatestRelease()
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to fetch latest release: %v\n", err)
		}
		return
	}

	latestVersion, err := normalizeVersion(release.TagName)
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Invalid version tag %q: %v\n", release.TagName, err)
		}
		return
	}

	cache.LastCheck = time.Now()
	cache.LatestVersion = latestVersion

	newer, err := isNewerVersion(latestVersion, currentVersion)
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Version comparison failed: %v\n", err)
		}
		return
	}

	if newer {
		printUpdateNotice(currentVersion, latestVersion)
		cache.NotifiedVersion = latestVersion
	}

	if err := saveCache(cacheDir, cache); err != nil && debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Failed to save update cache: %v\n", err)
	}
}

func fetchLatestRelease() (*GitHubRelease, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", GitHubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "devrag-update-checker")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if release.TagName == "" {
		return nil, fmt.Errorf("empty tag_name in response")
	}

	return &release, nil
}

// normalizeVersion extracts and validates semantic version from tag
func normalizeVersion(version string) (string, error) {
	matches := semverRegex.FindStringSubmatch(version)
	if matches == nil {
		return "", fmt.Errorf("invalid semver format: %q", version)
	}
	return fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3]), nil
}

// isNewerVersion compares two semantic versions
func isNewerVersion(latest, current string) (bool, error) {
	latestNorm, err := normalizeVersion(latest)
	if err != nil {
		return false, fmt.Errorf("latest version: %w", err)
	}

	currentNorm, err := normalizeVersion(current)
	if err != nil {
		return false, fmt.Errorf("current version: %w", err)
	}

	latestParts, err := parseVersion(latestNorm)
	if err != nil {
		return false, fmt.Errorf("parse latest: %w", err)
	}

	currentParts, err := parseVersion(currentNorm)
	if err != nil {
		return false, fmt.Errorf("parse current: %w", err)
	}

	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		if latestParts[i] > currentParts[i] {
			return true, nil
		}
		if latestParts[i] < currentParts[i] {
			return false, nil
		}
	}

	return len(latestParts) > len(currentParts), nil
}

func parseVersion(version string) ([]int, error) {
	parts := strings.Split(version, ".")
	result := make([]int, len(parts))

	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid version part %q: %w", part, err)
		}
		if num < 0 {
			return nil, fmt.Errorf("negative version part: %d", num)
		}
		result[i] = num
	}

	return result, nil
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

func printUpdateNotice(current, latest string) {
	msg := fmt.Sprintf("New version available: v%s (current: v%s)", latest, current)
	url := "https://github.com/tomohiro-owada/devrag/releases/latest"

	// Calculate box width
	width := len(msg)
	if len(url) > width {
		width = len(url)
	}
	width += 4 // padding

	// Build horizontal border
	border := strings.Repeat("═", width)

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "%s%s╔%s╗%s\n", colorBold, colorYellow, border, colorReset)
	fmt.Fprintf(os.Stderr, "%s%s║  %-*s  ║%s\n", colorBold, colorYellow, width-4, msg, colorReset)
	fmt.Fprintf(os.Stderr, "%s%s║  %s%-*s%s  %s║%s\n", colorBold, colorYellow, colorCyan, width-4, url, colorYellow, colorBold, colorReset)
	fmt.Fprintf(os.Stderr, "%s%s╚%s╝%s\n", colorBold, colorYellow, border, colorReset)
	fmt.Fprintf(os.Stderr, "\n")
}

func getCachePath(cacheDir string) (string, error) {
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}
		cacheDir = homeDir
	}
	return filepath.Join(cacheDir, CacheFileName), nil
}

func loadCache(cacheDir string) (*UpdateCache, error) {
	cache := &UpdateCache{}

	cachePath, err := getCachePath(cacheDir)
	if err != nil {
		return cache, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil // Not an error, just no cache yet
		}
		return cache, fmt.Errorf("read cache file: %w", err)
	}

	if err := json.Unmarshal(data, cache); err != nil {
		return cache, fmt.Errorf("parse cache file: %w", err)
	}

	return cache, nil
}

func saveCache(cacheDir string, cache *UpdateCache) error {
	cachePath, err := getCachePath(cacheDir)
	if err != nil {
		return err
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	return nil
}
