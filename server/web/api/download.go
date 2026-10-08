package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type fileReader struct {
	pos  int64
	size int64
	io.ReadSeeker
}

func newFR(size int64) *fileReader {
	return &fileReader{
		pos:  0,
		size: size,
	}
}

func (f *fileReader) Read(p []byte) (n int, err error) {
	if f.pos >= f.size {
		return 0, io.EOF
	}
	n = int(min(int64(len(p)), f.size-f.pos))
	// Always initialize the supplied buffer, including reused HTTP copy buffers.
	for i := 0; i < n; i++ {
		p[i] = byte((f.pos + int64(i)) * 31)
	}
	f.pos += int64(n)
	return n, nil
}

func (f *fileReader) Seek(offset int64, whence int) (int64, error) {
	pos := offset
	switch whence {
	case 0:
	case 1:
		pos += f.pos
	case 2:
		pos += f.size
	default:
		return f.pos, fmt.Errorf("invalid seek origin")
	}
	if pos < 0 {
		return f.pos, fmt.Errorf("negative seek")
	}
	f.pos = pos
	return f.pos, nil
}

// download godoc
//
//	@Summary		Generates test file of given size
//	@Description	Download the test file of given size (for speed testing purpose).
//
//	@Tags			API
//
//	@Param			size	path	string	true	"Test file size (in MB)"
//
//	@Produce		application/octet-stream
//	@Success		200 {file} file
//	@Router			/download/{size} [get]
func download(c *gin.Context) {
	szStr := c.Param("size")
	sz, err := strconv.ParseInt(szStr, 10, 64)
	if err != nil || sz < 1 || sz > (1<<63-1)>>20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "size must be a positive representable MiB count"})
		return
	}

	http.ServeContent(c.Writer, c.Request, fmt.Sprintln(szStr)+"mb.bin", time.Now(), newFR(sz<<20))
}
