package registry

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	PackagesDir = "packages"
	KeepFile    = ".gitkeep"
	IndexFile   = "index.json"
	CatalogFile = "catalog.md"
)

func FindRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", start, err)
	}
	for {
		if isRegularFile(filepath.Join(current, "go.mod")) && isDirectory(filepath.Join(current, PackagesDir)) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not find the repository root above %s: no ancestor contains go.mod and a packages directory", start)
		}
		current = parent
	}
}

func Load(root string) ([]Descriptor, error) {
	packagesDir := filepath.Join(root, PackagesDir)
	info, err := os.Lstat(packagesDir)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", packagesDir, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s is a symlink, the packages root must be a real directory", packagesDir)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", packagesDir)
	}
	paths, keep, err := scanPackages(packagesDir)
	if err != nil {
		return nil, err
	}
	descriptors := make([]Descriptor, 0, len(paths))
	for _, path := range paths {
		descriptor, err := loadDescriptor(packagesDir, path)
		if err != nil {
			return nil, err
		}
		descriptors = append(descriptors, descriptor)
	}
	keepPath := filepath.Join(packagesDir, KeepFile)
	if len(descriptors) == 0 && !keep {
		return nil, fmt.Errorf("%s is required while no descriptors are listed", keepPath)
	}
	if len(descriptors) > 0 && keep {
		return nil, fmt.Errorf("%s must be removed once descriptors are listed", keepPath)
	}
	if err := checkCollisions(descriptors); err != nil {
		return nil, err
	}
	return descriptors, nil
}

func scanPackages(dir string) ([]string, bool, error) {
	var paths []string
	keep := false
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink, symlinks are not allowed under %s", path, PackagesDir)
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("%s is not a regular file, special files are not allowed under %s", path, PackagesDir)
		}
		if path == dir {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("relativize %s: %w", path, err)
		}
		depth := strings.Count(filepath.ToSlash(rel), "/") + 1
		if entry.IsDir() {
			if depth != 1 {
				return fmt.Errorf("%s is an unexpected directory, the %s tree is %s/<owner>/<repo>.json", path, PackagesDir, PackagesDir)
			}
			return nil
		}
		switch depth {
		case 1:
			if rel == KeepFile {
				keep = true
				return nil
			}
			return fmt.Errorf("%s is unexpected, the %s directory holds only owner directories and %s", path, PackagesDir, KeepFile)
		case 2:
			if !strings.HasSuffix(entry.Name(), ".json") {
				return fmt.Errorf("%s is unexpected, descriptors are .json files", path)
			}
			if strings.TrimSuffix(entry.Name(), ".json") == "" {
				return fmt.Errorf("%s has an empty descriptor name", path)
			}
			paths = append(paths, path)
			if len(paths) > MaxDescriptors {
				return fmt.Errorf("more than %d descriptors found, the limit is %d", MaxDescriptors, MaxDescriptors)
			}
			return nil
		default:
			return fmt.Errorf("%s is an unexpected directory, the %s tree is %s/<owner>/<repo>.json", path, PackagesDir, PackagesDir)
		}
	})
	if err != nil {
		return nil, false, err
	}
	return paths, keep, nil
}

func loadDescriptor(packagesDir, path string) (Descriptor, error) {
	rel, err := filepath.Rel(packagesDir, path)
	if err != nil {
		return Descriptor{}, fmt.Errorf("relativize %s: %w", path, err)
	}
	display := filepath.ToSlash(filepath.Join(PackagesDir, rel))
	info, err := os.Lstat(path)
	if err != nil {
		return Descriptor{}, fmt.Errorf("stat %s: %w", path, err)
	}
	if info.Size() > MaxDescriptorBytes {
		return Descriptor{}, fmt.Errorf("%s: descriptor is %d bytes, the limit is %d bytes", display, info.Size(), MaxDescriptorBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Descriptor{}, fmt.Errorf("read %s: %w", path, err)
	}
	descriptor, err := ParseDescriptor(data)
	if err != nil {
		return Descriptor{}, fmt.Errorf("%s: %w", display, err)
	}
	stem := strings.TrimSuffix(filepath.ToSlash(rel), ".json")
	if stem != strings.ToLower(descriptor.Repository) {
		return Descriptor{}, fmt.Errorf("%s: descriptor path does not match the lowercase repository %q", display, descriptor.Repository)
	}
	return descriptor, nil
}

func checkCollisions(descriptors []Descriptor) error {
	seen := make(map[string]string, len(descriptors))
	for _, descriptor := range descriptors {
		lower := strings.ToLower(descriptor.Repository)
		if previous, ok := seen[lower]; ok {
			return fmt.Errorf("repositories %q and %q collide case insensitively", previous, descriptor.Repository)
		}
		seen[lower] = descriptor.Repository
	}
	return nil
}

func isRegularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func isDirectory(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}
