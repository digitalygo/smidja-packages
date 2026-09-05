package registry

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func Generate(root string) (int, error) {
	descriptors, err := Load(root)
	if err != nil {
		return 0, err
	}
	indexData := BuildIndex(descriptors)
	catalogData, err := BuildCatalog(descriptors)
	if err != nil {
		return 0, err
	}
	indexPath := filepath.Join(root, IndexFile)
	catalogPath := filepath.Join(root, CatalogFile)
	stagedIndex, err := stage(indexPath, indexData)
	if err != nil {
		return 0, err
	}
	stagedCatalog, err := stage(catalogPath, catalogData)
	if err != nil {
		_ = os.Remove(stagedIndex)
		return 0, err
	}
	if err := os.Rename(stagedIndex, indexPath); err != nil {
		_ = os.Remove(stagedIndex)
		_ = os.Remove(stagedCatalog)
		return 0, fmt.Errorf("replace %s: %w", indexPath, err)
	}
	if err := os.Rename(stagedCatalog, catalogPath); err != nil {
		_ = os.Remove(stagedCatalog)
		return 0, fmt.Errorf("replace %s: %w", catalogPath, err)
	}
	return len(descriptors), nil
}

func Check(root string) (int, error) {
	descriptors, err := Load(root)
	if err != nil {
		return 0, err
	}
	indexData := BuildIndex(descriptors)
	catalogData, err := BuildCatalog(descriptors)
	if err != nil {
		return 0, err
	}
	outputs := []struct {
		name string
		want []byte
	}{
		{IndexFile, indexData},
		{CatalogFile, catalogData},
	}
	var problems []string
	for _, output := range outputs {
		path := filepath.Join(root, output.name)
		got, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			problems = append(problems, fmt.Sprintf("%s is missing, run 'go run ./cmd/registry generate'", output.name))
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("read %s: %w", path, err)
		}
		if !bytes.Equal(got, output.want) {
			problems = append(problems, fmt.Sprintf("%s is stale, run 'go run ./cmd/registry generate'", output.name))
		}
	}
	if len(problems) > 0 {
		return 0, fmt.Errorf("committed outputs do not match the descriptors:\n%s", strings.Join(problems, "\n"))
	}
	return len(descriptors), nil
}

func stage(target string, data []byte) (string, error) {
	file, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".tmp-")
	if err != nil {
		return "", fmt.Errorf("stage %s: %w", target, err)
	}
	tempPath := file.Name()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("stage %s: %w", target, err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("stage %s: %w", target, err)
	}
	if err := os.Chmod(tempPath, 0o644); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("stage %s: %w", target, err)
	}
	return tempPath, nil
}
