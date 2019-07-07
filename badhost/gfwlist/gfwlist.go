package gfwlist

import (
	"bufio"
	"encoding/base64"
	"net/http"
	"regexp"
	"strings"
)

const gfwListURL = "https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt"

var cleanPatterns = []*regexp.Regexp{
	regexp.MustCompile("^\\|+"),
	regexp.MustCompile("https?://"),
	regexp.MustCompile("^\\."),
	regexp.MustCompile("^\\*.*?\\."),
}

// FetchBlockedDomains 下载黑名单
func FetchBlockedDomains() (domains []string, err error) {
	resp, err := http.Get(gfwListURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	decoder := base64.NewDecoder(base64.StdEncoding, resp.Body)
	scanner := bufio.NewScanner(decoder)

	scanner.Scan() // skip [AutoProxy 0.2.9]
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 || line[0] == '!' || line[0] == '@' {
			continue
		}

		if strings.IndexByte(line, '*') > 0 {
			continue
		}

		if strings.IndexByte(line, '\\') > 0 {
			continue
		}

		for _, p := range cleanPatterns {
			line = p.ReplaceAllString(line, "")
		}

		if p := strings.IndexByte(line, '/'); p > 0 {
			line = line[:p]
		}

		if strings.IndexByte(line, '.') > 0 {
			domains = append(domains, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return
}
