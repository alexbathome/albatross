package puttday

import "regexp"

var shareLinkPattern = regexp.MustCompile(`https://putt\.day/s/[A-Za-z0-9_-]+`)

// ExtractShareLink returns the first putt.day share link found in content, if any.
func ExtractShareLink(content string) (string, bool) {
	match := shareLinkPattern.FindString(content)
	return match, match != ""
}
