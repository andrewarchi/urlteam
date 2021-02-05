// Copyright (c) 2021 Andrew Archibald
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package tinytown processes URLTeam's second generation Terror of Tiny
// Town releases.
package tinytown

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/anacrolix/torrent"
)

// DownloadTinytown downloads all terroroftinytown releases via torrent.
func DownloadTinytown(dir string) error {
	ids, err := GetTinytownList()
	if err != nil {
		return err
	}

	conf := torrent.NewDefaultClientConfig()
	conf.DataDir = dir
	c, err := torrent.NewClient(conf)
	if err != nil {
		return err
	}

	for i, id := range ids {
		url := fmt.Sprintf("https://archive.org/download/%s/%s_archive.torrent", id, id)
		fmt.Printf("(%d/%d) Adding %s\n", i+1, len(ids), id)
		filename := filepath.Join(dir, path.Base(url))
		if err := saveFile(url, filename); err != nil {
			return err
		}

		t, err := c.AddTorrentFromFile(filename)
		if err != nil {
			return err
		}
		t.DownloadAll()
		if i%15 == 14 {
			c.WaitAll()
		}
	}
	c.WaitAll()
	return nil
}

const tinytownList = "https://archive.org/services/search/v1/scrape?q=subject:terroroftinytown&count=10000"

// GetTinytownList queries the Internet Archive for the identifiers of
// all incremental terroroftinytown releases.
func GetTinytownList() ([]string, error) {
	resp, err := http.Get(tinytownList)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	defer resp.Body.Close()

	type iaItem struct {
		Identifier string `json:"identifier"`
	}
	var items struct {
		Items []iaItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	ids := make([]string, len(items.Items))
	for i, item := range items.Items {
		ids[i] = item.Identifier
	}
	return ids, nil
}

func saveFile(url, filename string) error {
	if _, err := os.Stat(filename); err == nil {
		return nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %s", resp.Status)
	}
	defer resp.Body.Close()

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}