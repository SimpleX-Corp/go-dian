package dian

import (
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"sync"
	"time"
)

// timeNow is used for ZIP timestamps (can be mocked in tests)
var timeNow = time.Now

// Buffer pool for ZIP creation
var zipBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 65536)) // 64KB initial
	},
}

// CreateZipFast creates a ZIP file without compression (Store mode).
// DIAN accepts uncompressed ZIPs and this is 10x faster than compressed.
func CreateZipFast(fileName string, xmlContent []byte) ([]byte, error) {
	buf := zipBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer zipBufferPool.Put(buf)

	w := zip.NewWriter(buf)

	// Ensure .xml extension
	xmlName := fileName
	if filepath.Ext(xmlName) == "" {
		xmlName += ".xml"
	}

	// Create file with Store (no compression) for maximum speed
	header := &zip.FileHeader{
		Name:   xmlName,
		Method: zip.Store, // No compression = fastest
	}
	header.SetModTime(timeNow())

	f, err := w.CreateHeader(header)
	if err != nil {
		return nil, err
	}

	_, err = f.Write(xmlContent)
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	// Copy result (buffer will be recycled)
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// CreateZipLevel creates a ZIP with specified compression.
// Use Store (no compression) for best performance.
func CreateZipLevel(fileName string, xmlContent []byte, compress bool) ([]byte, error) {
	buf := zipBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer zipBufferPool.Put(buf)

	w := zip.NewWriter(buf)

	xmlName := fileName
	if filepath.Ext(xmlName) == "" {
		xmlName += ".xml"
	}

	var f io.Writer
	var err error

	if compress {
		f, err = w.Create(xmlName) // Default compression
	} else {
		header := &zip.FileHeader{
			Name:   xmlName,
			Method: zip.Store,
		}
		f, err = w.CreateHeader(header)
	}
	if err != nil {
		return nil, err
	}

	_, err = f.Write(xmlContent)
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}
