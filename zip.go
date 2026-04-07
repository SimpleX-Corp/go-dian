package dian

import (
	"archive/zip"
	"bytes"
	"path/filepath"
)

// CreateZip creates a ZIP file containing the given XML content.
// The fileName should be the base name without .zip extension.
// Returns the ZIP content ready to send to DIAN.
func CreateZip(fileName string, xmlContent []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Ensure .xml extension
	xmlName := fileName
	if filepath.Ext(xmlName) == "" {
		xmlName += ".xml"
	}

	f, err := w.Create(xmlName)
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

	return buf.Bytes(), nil
}

// ZipFileName generates the ZIP filename from invoice data.
// Format: NIT-DIAN_PREFIX-NUMBER.zip
// Example: 900123456-01-SETP990000001.zip
func ZipFileName(nit, prefix, number string) string {
	return nit + "-" + prefix + "-" + number + ".zip"
}
