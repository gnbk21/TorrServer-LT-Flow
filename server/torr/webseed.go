package torr

import (
	"errors"
	"net/url"
	"server/flow"
	"server/lt"
	"server/settings"
	"sort"
)

type WebSeedSummary struct {
	ID         string `json:"id"`
	Origin     string `json:"origin"`
	Disabled   bool   `json:"disabled"`
	AllowLocal bool   `json:"allow_local"`
}
type WebSeedStatus struct {
	Private bool             `json:"private"`
	Sources []WebSeedSummary `json:"sources"`
}

func sourceSettings(spec *TorrentSpec, metadata *lt.ParsedTorrent) map[string]settings.WebSeed {
	out := make(map[string]settings.WebSeed)
	for _, value := range metadata.WebSeeds {
		out[flow.WebSeedID(value)] = settings.WebSeed{URL: value}
	}
	for _, seed := range spec.WebSeeds {
		out[flow.WebSeedID(seed.URL)] = seed
	}
	return out
}

func TorrentWebSeeds(hash string) (WebSeedStatus, error) {
	out := WebSeedStatus{Sources: []WebSeedSummary{}}
	t := GetTorrentInfo(hash)
	if t == nil || t.TorrentSpec == nil {
		return out, errors.New("torrent not found")
	}
	t.mu.Lock()
	spec := *t.TorrentSpec
	t.mu.Unlock()
	metadata, err := lt.ParseTorrentBytes(spec.InfoBytes)
	if err != nil {
		return out, errors.New("wait for metadata or import original torrent")
	}
	out.Private = metadata.Private
	for id, seed := range sourceSettings(&spec, metadata) {
		u, e := url.Parse(seed.URL)
		if e != nil || u.Host == "" {
			continue
		}
		// Paths and signed queries may contain credentials. Only origin is shown.
		out.Sources = append(out.Sources, WebSeedSummary{ID: id, Origin: u.Scheme + "://" + u.Host, Disabled: seed.Disabled, AllowLocal: seed.AllowLocal})
	}
	sort.Slice(out.Sources, func(i, j int) bool { return out.Sources[i].ID < out.Sources[j].ID })
	return out, nil
}

func UpdateTorrentWebSeed(hashText, action, value, id string, allowLocal bool) error {
	if settings.ReadOnly {
		return errors.New("mirror management requires writable state")
	}
	hash := NewHashFromHex(hashText)
	if hash.IsZero() {
		return errors.New("invalid torrent")
	}
	t := GetTorrentInfo(hash.HexString())
	if t == nil || t.TorrentSpec == nil {
		return errors.New("torrent not found")
	}
	if t.LTHandle() == nil {
		t.mu.Lock()
		initial := *t.TorrentSpec
		t.mu.Unlock()
		if _, err := lt.ParseTorrentBytes(initial.InfoBytes); err != nil {
			return errors.New("known metadata required")
		}
		var err error
		t, err = NewTorrent(&initial, bts)
		if err != nil {
			return err
		}
	}
	t.sourceMu.Lock()
	defer t.sourceMu.Unlock()
	t.mu.Lock()
	spec := *t.TorrentSpec
	spec.WebSeeds = append([]settings.WebSeed(nil), spec.WebSeeds...)
	t.mu.Unlock()
	metadata, err := lt.ParseTorrentBytes(spec.InfoBytes)
	if err != nil {
		return errors.New("known metadata required")
	}
	seeds := sourceSettings(&spec, metadata)
	var seed settings.WebSeed
	switch action {
	case "add":
		if metadata.Private {
			return errors.New("additional sources are disabled for private torrents")
		}
		if _, err = flow.ValidateWebSeed(value, allowLocal); err != nil {
			return err
		}
		id = flow.WebSeedID(value)
		seed = settings.WebSeed{URL: value, AllowLocal: allowLocal}
		if _, exists := seeds[id]; !exists && len(seeds) >= 16 {
			return errors.New("mirror limit reached")
		}
	case "remove":
		var ok bool
		seed, ok = seeds[id]
		if !ok {
			return errors.New("mirror not found")
		}
		seed.Disabled = true
	default:
		return errors.New("unknown mirror action")
	}
	updated := false
	for i := range spec.WebSeeds {
		if flow.WebSeedID(spec.WebSeeds[i].URL) == id {
			spec.WebSeeds[i] = seed
			updated = true
			break
		}
	}
	if !updated {
		if len(spec.WebSeeds) >= 16 {
			return errors.New("mirror settings limit reached")
		}
		spec.WebSeeds = append(spec.WebSeeds, seed)
	}
	row := settings.GetTorrent(hash.HexString())
	if row == nil {
		t.mu.Lock()
		row = &settings.TorrentDB{Title: t.Title, Category: t.Category, Poster: t.Poster, Data: t.Data, Timestamp: t.Timestamp, Size: t.Size}
		t.mu.Unlock()
	}
	row.TorrentSpec = settingsSpec(&spec)
	if err = settings.WriteLibraryBatch([]*settings.TorrentDB{row}, false); err != nil {
		return errors.New("cannot persist mirror settings")
	}
	t.mu.Lock()
	t.TorrentSpec.WebSeeds = spec.WebSeeds
	t.mu.Unlock()
	if bts != nil && bts.preparation != nil {
		p := bts.preparation
		p.mu.Lock()
		changed := false
		for _, j := range p.jobs {
			if j.Spec.InfoHash == hash {
				j.Spec.WebSeeds = append([]settings.WebSeed(nil), spec.WebSeeds...)
				changed = true
			}
		}
		var e error
		if changed {
			e = p.save()
		}
		p.mu.Unlock()
		if e != nil {
			return errors.New("mirror saved; cannot update preparation state")
		}
	}
	handle := t.LTHandle()
	if handle == nil {
		t, err = NewTorrent(&spec, bts)
		if err != nil {
			return err
		}
		handle = t.LTHandle()
	}
	if err = handle.SetURLSeed(seed.URL, seed.Disabled, seed.AllowLocal); err != nil {
		return errors.New("mirror saved; native source update unavailable")
	}
	return nil
}
