package torrfs

import (
	"fmt"
	"io/fs"
	"time"

	"server/settings"

	"server/torr"
)

type CategoryDir struct {
	info fs.FileInfo
}

func NewCategoryDir(category string) *CategoryDir {
	category = categoryName(category)
	d := &CategoryDir{
		info: info{
			name:  category,
			size:  4096,
			mode:  0o555,
			mtime: time.Unix(477033666, 0),
			isDir: true,
		},
	}
	return d
}

func categoryName(category string) string {
	category = SanitizeName(category)
	if category == "" {
		return "other"
	}
	return category
}

func torrentName(t *torr.Torrent) string {
	name := SanitizeName(t.Title)
	if name == "" {
		name = t.Hash().String()
	}
	return name
}

func (d *CategoryDir) Stat() (fs.FileInfo, error) {
	return d.info, nil
}

// torrentNames preserves unique titles and disambiguates duplicate sanitized
// names with the shortest unique hash prefix, starting at eight hex digits.
// Reserve literal titles too, so a generated suffix cannot hide another torrent.
// Names depend on the visible set, not its listing order. Adding a duplicate
// necessarily changes a formerly unique title into a suffixed directory.
func torrentNames(torrs []*torr.Torrent) []string {
	names := make([]string, len(torrs))
	hashes := make([]string, len(torrs))
	counts := make(map[string]int, len(torrs))
	for i, t := range torrs {
		names[i], hashes[i] = torrentName(t), t.Hash().HexString()
		counts[names[i]]++
	}
	prefixes := make(map[string]int)
	for i, name := range names {
		if counts[name] < 2 {
			continue
		}
		for width := 8; width <= len(hashes[i]); width += 4 {
			prefixes[name+"\x00"+hashes[i][:width]]++
		}
	}
	used := make(map[string]bool, len(torrs))
	for name := range counts {
		used[name] = true
	}
	for i, name := range names {
		if counts[name] < 2 {
			continue
		}
		candidate := ""
		for width := 8; width <= len(hashes[i]); width += 4 {
			prefix := hashes[i][:width]
			candidate = name + " [" + prefix + "]"
			if prefixes[name+"\x00"+prefix] == 1 && !used[candidate] {
				break
			}
		}
		// Full hashes can also occur in a literal title. Duplicate identities
		// cannot occur in the torrent registry, but this bounds that case too.
		base := candidate
		for ordinal := 2; used[candidate]; ordinal++ {
			candidate = fmt.Sprintf("%s (%d)", base, ordinal)
		}
		names[i], used[candidate] = candidate, true
	}
	return names
}

func (d *CategoryDir) ReadDir(n int) ([]fs.DirEntry, error) {
	nodes := []fs.DirEntry{}
	var torrs []*torr.Torrent
	for _, t := range torr.ListTorrent() {
		if categoryName(t.Category) != d.Name() {
			continue
		}
		if settings.BTsets().ShowFSActiveTorr && !t.GotInfo() {
			continue
		}
		torrs = append(torrs, t)
	}
	for i, name := range torrentNames(torrs) {
		nodes = append(nodes, NewTorrDir(nil, name, torrs[i]))
	}

	return nodes, nil
}

// INode
func (d *CategoryDir) Open(name string) (fs.File, error) { return Open(d, name) }
func (d *CategoryDir) Parent() INode                     { return nil }
func (d *CategoryDir) Torrent() *torr.Torrent            { return nil }
func (d *CategoryDir) SetTorrent(torr *torr.Torrent)     {}

// DirEntry
func (d *CategoryDir) Name() string { return d.info.Name() }
func (d *CategoryDir) IsDir() bool  { return true }
func (d *CategoryDir) Type() fs.FileMode {
	s, _ := d.Stat()
	return s.Mode()
}
func (d *CategoryDir) Info() (fs.FileInfo, error) { return d.info, nil }

// File
func (d *CategoryDir) Read(bytes []byte) (int, error) { return 0, fs.ErrInvalid }
func (d *CategoryDir) Close() error                   { return nil }
