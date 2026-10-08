package flow

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sort"
	"time"
)

var errIndex = errors.New("resident index unavailable or outside inspection limits")

type IndexPoint struct {
	Offset  int64
	Seconds float64
}
type BurstIndex struct {
	Source string
	Points []IndexPoint
}

// Rate is a coarse inter-index byte/time slope around a file-local position.
// It includes multiplexed bytes, never claims an instantaneous decoder bitrate,
// and cannot lower average demand or raise it beyond twice the credible average.
func (b BurstIndex) Rate(offset int64, average float64) float64 {
	if average <= 0 || math.IsNaN(average) || math.IsInf(average, 0) || len(b.Points) < 2 {
		return average
	}
	i := sort.Search(len(b.Points), func(i int) bool { return b.Points[i].Offset > offset })
	if i == 0 || i == len(b.Points) {
		return average
	}
	a, z := b.Points[i-1], b.Points[i]
	duration := z.Seconds - a.Seconds
	if duration < .5 || duration > 30 {
		return average
	}
	return max(average, min(2*average, float64(z.Offset-a.Offset)/duration))
}

type indexReader struct {
	r               io.ReaderAt
	size, remaining int64
	until           time.Time
	headers         int
}

func (r *indexReader) read(off, n int64) ([]byte, error) {
	if off < 0 || n < 0 || off > r.size || n > r.size-off || n > r.remaining || time.Now().After(r.until) {
		return nil, errIndex
	}
	r.remaining -= n
	b := make([]byte, n)
	got, err := r.r.ReadAt(b, off)
	if int64(got) != n || (err != nil && err != io.EOF) {
		return nil, errIndex
	}
	return b, nil
}

// ReadBurstIndex inspects headers and referenced indexes only. The supplied
// reader must be resident-only: a cache miss fails immediately. CPU/wall time,
// bytes, header count, table entries and retained points are all bounded.
func ReadBurstIndex(reader io.ReaderAt, size int64) (BurstIndex, error) {
	r := &indexReader{r: reader, size: size, remaining: 4 * MiB, until: time.Now().Add(150 * time.Millisecond)}
	h, err := r.read(0, 8)
	if err != nil {
		return BurstIndex{}, err
	}
	var index BurstIndex
	if binary.BigEndian.Uint32(h) == 0x1a45dfa3 {
		index, err = r.matroska()
	} else if string(h[4:8]) == "ftyp" {
		index, err = r.mp4()
	} else {
		err = errIndex
	}
	if err != nil || len(index.Points) < 2 || len(index.Points) > 4096 {
		return BurstIndex{}, errIndex
	}
	for i, point := range index.Points {
		if point.Offset < 0 || point.Offset >= size || point.Seconds < 0 || point.Seconds > 30*86400 || math.IsNaN(point.Seconds) || math.IsInf(point.Seconds, 0) {
			return BurstIndex{}, errIndex
		}
		if i > 0 && (point.Offset <= index.Points[i-1].Offset || point.Seconds <= index.Points[i-1].Seconds) {
			return BurstIndex{}, errIndex
		}
	}
	return index, nil
}

type element struct {
	id         uint64
	start, end int64
}

func (r *indexReader) ebml(off, end int64) (element, error) {
	r.headers++
	if r.headers > 32768 {
		return element{}, errIndex
	}
	vint := func(pos int64, id bool) (uint64, int64, error) {
		b, err := r.read(pos, 1)
		if err != nil {
			return 0, 0, err
		}
		mask, n := byte(0x80), int64(1)
		for mask > 0 && b[0]&mask == 0 {
			mask >>= 1
			n++
		}
		if mask == 0 || (id && n > 4) || pos+n > end {
			return 0, 0, errIndex
		}
		data, err := r.read(pos, n)
		if err != nil {
			return 0, 0, err
		}
		value := uint64(data[0])
		if !id {
			value &= uint64(mask - 1)
		}
		for _, x := range data[1:] {
			value = value<<8 | uint64(x)
		}
		return value, n, nil
	}
	id, a, err := vint(off, true)
	if err != nil {
		return element{}, err
	}
	n, b, err := vint(off+a, false)
	if err != nil {
		return element{}, err
	}
	start := off + a + b
	if n == uint64(1)<<(uint(b)*7)-1 && id == 0x18538067 {
		n = uint64(end - start)
	}
	if start > end || n > uint64(end-start) {
		return element{}, errIndex
	}
	return element{id, start, start + int64(n)}, nil
}
func (r *indexReader) uint(e element) (uint64, error) {
	if e.end-e.start < 1 || e.end-e.start > 8 {
		return 0, errIndex
	}
	b, err := r.read(e.start, e.end-e.start)
	var n uint64
	for _, x := range b {
		n = n<<8 | uint64(x)
	}
	return n, err
}
func (r *indexReader) children(start, end int64, visit func(element) error) error {
	for pos := start; pos < end; {
		e, err := r.ebml(pos, end)
		if err != nil {
			return err
		}
		if err = visit(e); err != nil {
			return err
		}
		pos = e.end
	}
	return nil
}
func (r *indexReader) matroska() (BurstIndex, error) {
	header, err := r.ebml(0, r.size)
	if err != nil {
		return BurstIndex{}, err
	}
	segment, err := r.ebml(header.end, r.size)
	if err != nil || segment.id != 0x18538067 {
		return BurstIndex{}, errIndex
	}
	scale := uint64(1000000)
	infoKnown := false
	var cues *element
	refs := make(map[uint64]int64)
	readInfo := func(e element) error {
		infoKnown = true
		return r.children(e.start, e.end, func(child element) error {
			if child.id == 0x2ad7b1 {
				value, err := r.uint(child)
				if err != nil || value == 0 || value > 1000000000 {
					return errIndex
				}
				scale = value
			}
			return nil
		})
	}
	for pos, count := segment.start, 0; pos < segment.end && count < 64; count++ {
		e, err := r.ebml(pos, segment.end)
		if err != nil {
			return BurstIndex{}, err
		}
		switch e.id {
		case 0x1549a966:
			if err := readInfo(e); err != nil {
				return BurstIndex{}, err
			}
		case 0x1c53bb6b:
			copy := e
			cues = &copy
		case 0x114d9b74:
			if err := r.children(e.start, e.end, func(seek element) error {
				if seek.id != 0x4dbb {
					return nil
				}
				var id, off uint64
				if err := r.children(seek.start, seek.end, func(item element) error {
					if item.id != 0x53ab && item.id != 0x53ac {
						return nil
					}
					value, err := r.uint(item)
					if item.id == 0x53ab {
						id = value
					} else {
						off = value
					}
					return err
				}); err != nil {
					return err
				}
				if (id == 0x1c53bb6b || id == 0x1549a966) && off < uint64(segment.end-segment.start) {
					refs[id] = segment.start + int64(off)
				}
				return nil
			}); err != nil {
				return BurstIndex{}, err
			}
		}
		pos = e.end
		if cues != nil {
			break
		}
		if off, ok := refs[0x1c53bb6b]; ok {
			e, err := r.ebml(off, segment.end)
			if err != nil || e.id != 0x1c53bb6b {
				return BurstIndex{}, errIndex
			}
			cues = &e
			break
		}
	}
	if off, ok := refs[0x1549a966]; ok {
		e, err := r.ebml(off, segment.end)
		if err != nil || e.id != 0x1549a966 {
			return BurstIndex{}, errIndex
		}
		if err = readInfo(e); err != nil {
			return BurstIndex{}, err
		}
	}
	if cues == nil || !infoKnown {
		return BurstIndex{}, errIndex
	}
	index := BurstIndex{Source: "matroska-cues-coarse"}
	err = r.children(cues.start, cues.end, func(cue element) error {
		if cue.id != 0xbb {
			return nil
		}
		if len(index.Points) >= 4096 {
			return errIndex
		}
		var ticks, offset uint64
		haveTime, haveOffset := false, false
		err := r.children(cue.start, cue.end, func(item element) error {
			if item.id == 0xb3 {
				value, err := r.uint(item)
				ticks, haveTime = value, err == nil
				return err
			}
			if item.id == 0xb7 {
				return r.children(item.start, item.end, func(position element) error {
					if position.id == 0xf1 {
						value, err := r.uint(position)
						if !haveOffset || value < offset {
							offset = value
						}
						haveOffset = err == nil
						return err
					}
					return nil
				})
			}
			return nil
		})
		if err != nil || !haveTime || !haveOffset || offset >= uint64(segment.end-segment.start) {
			return errIndex
		}
		index.Points = append(index.Points, IndexPoint{segment.start + int64(offset), float64(ticks) * float64(scale) / 1e9})
		return nil
	})
	return index, err
}

type mp4box struct {
	kind       string
	start, end int64
}

func (r *indexReader) boxes(start, end int64) ([]mp4box, error) {
	var out []mp4box
	for pos := start; pos < end; {
		r.headers++
		if r.headers > 256 {
			return nil, errIndex
		}
		h, err := r.read(pos, 8)
		if err != nil {
			return nil, err
		}
		n := uint64(binary.BigEndian.Uint32(h))
		head := int64(8)
		if n == 1 {
			b, err := r.read(pos+8, 8)
			if err != nil {
				return nil, err
			}
			n, head = binary.BigEndian.Uint64(b), 16
		}
		if n == 0 {
			n = uint64(end - pos)
		}
		if n < uint64(head) || n > uint64(end-pos) {
			return nil, errIndex
		}
		out = append(out, mp4box{string(h[4:8]), pos + head, pos + int64(n)})
		pos += int64(n)
	}
	return out, nil
}
func (r *indexReader) mp4() (BurstIndex, error) {
	top, err := r.boxes(0, r.size)
	if err != nil {
		return BurstIndex{}, err
	}
	for _, box := range top {
		if box.kind != "moov" {
			continue
		}
		tracks, err := r.boxes(box.start, box.end)
		if err != nil {
			return BurstIndex{}, err
		}
		for _, track := range tracks {
			if track.kind != "trak" {
				continue
			}
			index, err := r.mp4Track(track)
			if err == nil {
				return index, nil
			}
		}
	}
	return BurstIndex{}, errIndex
}
func (r *indexReader) mp4Track(track mp4box) (BurstIndex, error) {
	tables := make(map[string][]byte)
	var walk func(mp4box, int) error
	walk = func(parent mp4box, depth int) error {
		if depth > 4 {
			return errIndex
		}
		boxes, err := r.boxes(parent.start, parent.end)
		if err != nil {
			return err
		}
		for _, box := range boxes {
			switch box.kind {
			case "mdia", "minf", "stbl":
				if err := walk(box, depth+1); err != nil {
					return err
				}
			case "mdhd", "hdlr", "stts", "stsc", "stco", "co64":
				if _, exists := tables[box.kind]; exists {
					return errIndex
				}
				b, err := r.read(box.start, box.end-box.start)
				if err != nil {
					return err
				}
				tables[box.kind] = b
			}
		}
		return nil
	}
	if err := walk(track, 0); err != nil {
		return BurstIndex{}, err
	}
	handler, mdhd := tables["hdlr"], tables["mdhd"]
	if len(handler) < 12 || string(handler[8:12]) != "vide" || len(mdhd) < 16 {
		return BurstIndex{}, errIndex
	}
	scaleOffset := 12
	if mdhd[0] == 1 {
		scaleOffset = 20
	} else if mdhd[0] != 0 {
		return BurstIndex{}, errIndex
	}
	if len(mdhd) < scaleOffset+4 {
		return BurstIndex{}, errIndex
	}
	scale := binary.BigEndian.Uint32(mdhd[scaleOffset:])
	if scale == 0 {
		return BurstIndex{}, errIndex
	}
	entries := func(name string, width int) ([]byte, int, error) {
		b := tables[name]
		if len(b) < 8 || b[0] != 0 {
			return nil, 0, errIndex
		}
		count := int(binary.BigEndian.Uint32(b[4:8]))
		if count < 1 || count > 131072 || len(b)-8 != count*width {
			return nil, 0, errIndex
		}
		return b[8:], count, nil
	}
	stts, nt, err := entries("stts", 8)
	if err != nil {
		return BurstIndex{}, err
	}
	stsc, ns, err := entries("stsc", 12)
	if err != nil {
		return BurstIndex{}, err
	}
	name, width := "stco", 4
	if tables["co64"] != nil {
		name, width = "co64", 8
	}
	offsets, count, err := entries(name, width)
	if err != nil {
		return BurstIndex{}, err
	}
	u32 := func(b []byte, at int) uint64 { return uint64(binary.BigEndian.Uint32(b[at:])) }
	if u32(stsc, 0) != 1 {
		return BurstIndex{}, errIndex
	}
	for i := 0; i < ns; i++ {
		if u32(stsc, i*12) > uint64(count) || u32(stsc, i*12+4) == 0 || u32(stsc, i*12+8) == 0 || (i > 0 && u32(stsc, i*12) <= u32(stsc, (i-1)*12)) {
			return BurstIndex{}, errIndex
		}
	}
	index := BurstIndex{Source: "mp4-video-chunks-coarse"}
	stride := max(1, (count+4095)/4096)
	run, mapping := 0, 0
	remaining := u32(stts, 0)
	var ticks uint64
	for chunk := 0; chunk < count; chunk++ {
		if time.Now().After(r.until) {
			return BurstIndex{}, errIndex
		}
		for mapping+1 < ns && u32(stsc, (mapping+1)*12) <= uint64(chunk+1) {
			mapping++
		}
		offset := u32(offsets, chunk*width)
		if width == 8 {
			offset = binary.BigEndian.Uint64(offsets[chunk*width:])
		}
		if offset >= uint64(r.size) {
			return BurstIndex{}, errIndex
		}
		if chunk%stride == 0 {
			index.Points = append(index.Points, IndexPoint{int64(offset), float64(ticks) / float64(scale)})
		}
		need := u32(stsc, mapping*12+4)
		for need > 0 {
			if run >= nt || remaining == 0 {
				return BurstIndex{}, errIndex
			}
			take := min(need, remaining)
			delta := u32(stts, run*8+4)
			if delta == 0 || take*delta > uint64(30*86400)*uint64(scale)-min(ticks, uint64(30*86400)*uint64(scale)) {
				return BurstIndex{}, errIndex
			}
			ticks += take * delta
			need -= take
			remaining -= take
			if remaining == 0 {
				run++
				if run < nt {
					remaining = u32(stts, run*8)
				}
			}
		}
	}
	if run != nt || remaining != 0 {
		return BurstIndex{}, errIndex
	}
	return index, nil
}
