package flow

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func atom(kind string, body ...[]byte) []byte {
	data := bytes.Join(body, nil)
	h := make([]byte, 8)
	binary.BigEndian.PutUint32(h, uint32(len(data)+8))
	copy(h[4:], kind)
	return append(h, data...)
}
func words(values ...uint32) []byte {
	b := make([]byte, 4*len(values))
	for i, v := range values {
		binary.BigEndian.PutUint32(b[i*4:], v)
	}
	return b
}
func mp4IndexFixture(co64 bool) []byte {
	mdhd := words(0, 0, 0, 1000, 4000)
	hdlr := append(words(0, 0), []byte("vide")...)
	stts, stsc := words(0, 1, 4, 1000), words(0, 1, 1, 1, 1)
	name, offsets := "stco", words(0, 4, 100, 200, 400, 800)
	if co64 {
		name, offsets = "co64", words(0, 4, 0, 100, 0, 200, 0, 400, 0, 800)
	}
	return bytes.Join([][]byte{atom("ftyp", []byte("isom")), atom("mdat", make([]byte, 1024)),
		atom("moov", atom("trak", atom("mdia", atom("mdhd", mdhd), atom("hdlr", hdlr), atom("minf", atom("stbl", atom("stts", stts), atom("stsc", stsc), atom(name, offsets))))))}, nil)
}
func ebmlElement(id []byte, body ...[]byte) []byte {
	data := bytes.Join(body, nil)
	size := []byte{byte(len(data)) | 0x80}
	if len(data) >= 127 {
		size = []byte{byte(len(data)>>8) | 0x40, byte(len(data))}
	}
	return bytes.Join([][]byte{id, size, data}, nil)
}
func cue(ticks, offset uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, ticks)
	p := make([]byte, 2)
	binary.BigEndian.PutUint16(p, offset)
	return ebmlElement([]byte{0xbb}, ebmlElement([]byte{0xb3}, b), ebmlElement([]byte{0xb7}, ebmlElement([]byte{0xf1}, p)))
}
func TestResidentIndexesAndCoarseBounds(t *testing.T) {
	for _, co64 := range []bool{false, true} {
		data := mp4IndexFixture(co64)
		index, err := ReadBurstIndex(bytes.NewReader(data), int64(len(data)))
		if err != nil || len(index.Points) != 4 || index.Points[2].Seconds != 2 {
			t.Fatal(index, err)
		}
		if got := index.Rate(500, 100); got != 200 {
			t.Fatal("burst cap", got)
		}
		if got := index.Rate(100, 150); got != 150 {
			t.Fatal("hint lowered demand", got)
		}
		if got := index.Rate(10000, 150); got != 150 {
			t.Fatal("extrapolated missing interval", got)
		}
	}
	data := ebmlElement([]byte{0x1a, 0x45, 0xdf, 0xa3})
	segment := ebmlElement([]byte{0x15, 0x49, 0xa9, 0x66}, ebmlElement([]byte{0x2a, 0xd7, 0xb1}, []byte{0x0f, 0x42, 0x40}))
	segment = append(segment, ebmlElement([]byte{0x1c, 0x53, 0xbb, 0x6b}, cue(0, 100), cue(1000, 200), cue(2000, 400))...)
	segment = append(segment, ebmlElement([]byte{0xec}, make([]byte, 1024))...)
	data = append(data, ebmlElement([]byte{0x18, 0x53, 0x80, 0x67}, segment)...)
	index, err := ReadBurstIndex(bytes.NewReader(data), int64(len(data)))
	if err != nil || index.Source != "matroska-cues-coarse" || len(index.Points) != 3 || index.Points[1].Seconds != 1 {
		t.Fatal(index, err)
	}
}
func TestMalformedAndMissingIndexesFallBack(t *testing.T) {
	data := mp4IndexFixture(false)
	for _, candidate := range [][]byte{nil, data[:8], bytes.Repeat([]byte{0xff}, 1024), append([]byte{0, 0, 0, 1, 'f', 't', 'y', 'p'}, bytes.Repeat([]byte{0xff}, 8)...)} {
		if _, err := ReadBurstIndex(bytes.NewReader(candidate), int64(len(candidate))); err == nil {
			t.Fatal("malformed index accepted")
		}
	}
	changed := bytes.Clone(data)
	pos := bytes.Index(changed, []byte("stts"))
	binary.BigEndian.PutUint32(changed[pos+8:], 0xffffffff)
	if _, err := ReadBurstIndex(bytes.NewReader(changed), int64(len(changed))); err == nil {
		t.Fatal("unbounded table accepted")
	}
	changed = bytes.Clone(data)
	pos = bytes.Index(changed, []byte("stco"))
	binary.BigEndian.PutUint32(changed[pos+16:], 50)
	if _, err := ReadBurstIndex(bytes.NewReader(changed), int64(len(changed))); err == nil {
		t.Fatal("backward offsets accepted")
	}
}
func FuzzResidentIndex(f *testing.F) {
	f.Add(mp4IndexFixture(false))
	f.Add([]byte{0x1a, 0x45, 0xdf, 0xa3, 0x80})
	f.Fuzz(func(t *testing.T, data []byte) {
		if int64(len(data)) > 4*MiB {
			t.Skip()
		}
		_, _ = ReadBurstIndex(bytes.NewReader(data), int64(len(data)))
	})
}
