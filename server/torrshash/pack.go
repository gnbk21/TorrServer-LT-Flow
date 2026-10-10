package torrshash

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

const maxTokenBytes = 1 << 20
const maxTokenFields = 256
const maxEncodedTokenBytes = 24 << 10
const maxCompressedTokenBytes = 16 << 10

func Pack(hash *TorrsHash) (string, error) {
	data, err := pack(hash)
	if err != nil {
		return "", err
	}
	return Encode62(data), nil
}

func PackBytes(hash *TorrsHash) ([]byte, error) {
	return pack(hash)
}

func Unpack(token string) (*TorrsHash, error) {
	token = strings.TrimSpace(token)
	if len(token) > maxEncodedTokenBytes || !IsBase62(token) {
		return nil, errors.New("invalid or oversized torrent token")
	}
	data := Decode62(token)
	return UnpackBytes(data)
}

func UnpackBytes(data []byte) (*TorrsHash, error) {
	return unpack(data)
}

func pack(t *TorrsHash) ([]byte, error) {
	if t == nil || len(t.Fields) > maxTokenFields {
		return nil, errors.New("invalid torrent token")
	}
	hashBytes, err := hex.DecodeString(strings.TrimSpace(t.Hash))
	if err != nil || len(hashBytes) != 20 {
		return nil, errors.New("compact torrent tokens require a v1 SHA-1 identity; use a magnet for v2")
	}
	total := 20
	for _, f := range t.Fields {
		if f == nil {
			return nil, errors.New("nil torrent token field")
		}
		total += 3 + len(f.Value)
		if total > maxTokenBytes {
			return nil, errors.New("torrent token exceeds limit")
		}
	}
	buf := new(bytes.Buffer)
	zw, err := zlib.NewWriterLevel(buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}

	_, err = zw.Write(hashBytes)
	if err != nil {
		return nil, err
	}

	for _, f := range t.Fields {
		err = f.write(zw)
		if err != nil {
			return nil, err
		}
	}

	err = zw.Close()
	if buf.Len() > maxCompressedTokenBytes {
		return nil, errors.New("compressed torrent token exceeds limit")
	}
	return buf.Bytes(), err
}

func unpack(data []byte) (*TorrsHash, error) {
	if len(data) > maxCompressedTokenBytes {
		return nil, errors.New("torrent token exceeds limit")
	}
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	reader := &io.LimitedReader{R: zr, N: maxTokenBytes + 1}

	hashBuf := make([]byte, 20)
	if _, err = io.ReadFull(reader, hashBuf); err != nil {
		return nil, err
	}

	th := &TorrsHash{}
	th.Hash = hex.EncodeToString(hashBuf)

	for {
		field, err := ReadField(reader)
		if reader.N <= 0 {
			return nil, errors.New("expanded torrent token exceeds limit")
		}
		if err == nil && field == nil {
			//End on read
			return th, nil
		}
		if err != nil {
			return th, err
		}

		if len(th.Fields) >= maxTokenFields {
			return nil, errors.New("too many torrent token fields")
		}
		th.Fields = append(th.Fields, field)
	}
}
