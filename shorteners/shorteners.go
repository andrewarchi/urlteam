// Copyright (c) 2021 Andrew Archibald
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package shorteners

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/andrewarchi/urlhero/ia"
)

type Shortener struct {
	Name         string
	Host         string
	Prefix       string
	Alphabet     string
	Pattern      *regexp.Regexp
	CleanFunc    CleanFunc
	IsVanityFunc IsVanityFunc
}

type CleanFunc func(shortcode string, u *url.URL) string
type IsVanityFunc func(shortcode string) bool

var Shorteners = []*Shortener{
	Allst,
	Debli,
	Qrcx,
	Redht,
	Uconn,
}

// Clean extracts the shortcode from a URL. An empty string is returned
// when no shortcode can be found.
func (s *Shortener) Clean(shortURL string) (string, error) {
	u, err := url.Parse(shortURL)
	if err != nil {
		return "", err
	}
	return cleanURL(u, s.CleanFunc), nil
}

// CleanURL extracts the shortcode from a URL. An empty string is
// returned when no shortcode can be found.
func (s *Shortener) CleanURL(u *url.URL) string {
	return cleanURL(u, s.CleanFunc)
}

func cleanURL(u *url.URL, clean func(shortcode string, u *url.URL) string) string {
	shortcode := strings.TrimPrefix(u.Path, "/")
	// Exclude placeholders:
	//   https://deb.li/<key>
	//   https://deb.li/<name>
	if len(shortcode) >= 2 && shortcode[0] == '<' && shortcode[len(shortcode)-1] == '>' {
		return ""
	}
	// Remove trailing junk:
	//   http://a.ll.st/Instagram","isCrawlable":true,"thumbnail
	//   http://qr.cx/plvd]http:/qr.cx/plvd[/link]
	//   http://qr.cx/)
	//   https://red.ht/sig>
	//   https://red.ht/1zzgkXp&esheet=51687448&newsitemid=20170921005271&lan=en-US&anchor=Red+Hat+blog&index=5&md5=7ea962d15a0e5bf8e35f385550f4decb
	//   https://red.ht/13LslKt&quot
	//   https://red.ht/2k3DNz3’
	//   https://red.ht/21Krw4z%C2%A0   (nbsp)
	if i := strings.IndexAny(shortcode, "\"])>&’\u00a0"); i != -1 {
		shortcode = shortcode[:i]
	}
	shortcode = strings.TrimSuffix(shortcode, "/")
	if shortcode == "" {
		return ""
	}
	if clean != nil {
		shortcode = clean(shortcode, u)
	}
	shortcode = strings.TrimSuffix(shortcode, "/")
	switch shortcode {
	case "favicon.ico", "robots.txt":
		return ""
	}
	return shortcode
}

// CleanURLs extracts, deduplicates, and sorts the shortcodes in slice
// of URLs.
func (s *Shortener) CleanURLs(urls []string) ([]string, error) {
	shortcodesMap := make(map[string]struct{})
	var shortcodes []string
	for _, shortURL := range urls {
		shortcode, err := s.Clean(shortURL)
		if err != nil {
			return nil, err
		} else if shortcode == "" {
			continue
		}
		if s.Pattern != nil && !s.Pattern.MatchString(shortcode) {
			return nil, fmt.Errorf("%s: shortcode %q does not match alphabet %s after cleaning: %q", s.Name, shortcode, s.Pattern, shortURL)
		}
		if _, ok := shortcodesMap[shortcode]; !ok {
			shortcodesMap[shortcode] = struct{}{}
			shortcodes = append(shortcodes, shortcode)
		}
	}
	s.Sort(shortcodes)
	return shortcodes, nil
}

// IsVanity returns true when a shortcode is a vanity code. There are
// many false negatives for vanity codes that are programmatically
// indistinguishable from generated codes.
func (s *Shortener) IsVanity(shortcode string) bool {
	return s.IsVanityFunc != nil && s.IsVanityFunc(shortcode)
}

// Sort sorts shorter codes first and generated codes before vanity
// codes.
func (s *Shortener) Sort(shortcodes []string) {
	less := func(a, b string) bool {
		return (len(a) == len(b) && a < b) || len(a) < len(b)
	}
	if s.IsVanityFunc != nil {
		less = func(a, b string) bool {
			aVanity := s.IsVanityFunc(a)
			bVanity := s.IsVanityFunc(b)
			return (aVanity == bVanity && ((len(a) == len(b) && a < b) || len(a) < len(b))) ||
				(!aVanity && bVanity)
		}
	}
	sort.Slice(shortcodes, func(i, j int) bool {
		return less(shortcodes[i], shortcodes[j])
	})
}

// GetIAShortcodes queries all the shortcodes that have been archived on
// the Internet Archive.
func (s *Shortener) GetIAShortcodes() ([]string, error) {
	timemap, err := ia.GetTimemap(s.Host, &ia.TimemapOptions{
		Collapse:    "original",
		Fields:      []string{"original"},
		MatchPrefix: true,
		Limit:       100000,
	})
	if err != nil {
		return nil, err
	}
	urls := make([]string, len(timemap))
	for i, link := range timemap {
		urls[i] = link[0]
	}
	return s.CleanURLs(urls)
}
