package connectors

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// createZipFromDir creates a ZIP archive at zipPath from the contents of srcDir.
// It uses FileInfoHeader to preserve proper file modes (permissions).
func createZipFromDir(t *testing.T, srcDir, zipPath string) {
	t.Helper()

	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip file: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		if info.IsDir() {
			header.Name += "/"
		}

		fw, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = fw.Write(data)
		return err
	})
	if err != nil {
		t.Fatalf("walk src dir: %v", err)
	}
}
