package inventur

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyDir(t *testing.T) {
	// Create a temporary source directory with some content
	srcDir := t.TempDir()

	// Create a regular file
	err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("hello"), 0644)
	assert.NoError(t, err)

	// Create a nested directory with a file
	nestedDir := filepath.Join(srcDir, "nested")
	err = os.Mkdir(nestedDir, 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(nestedDir, "file2.txt"), []byte("world"), 0644)
	assert.NoError(t, err)

	// Define destination directory (should not exist yet)
	dstDir := filepath.Join(t.TempDir(), "dest_dir")

	// Call the function
	err = copyDir(srcDir, dstDir)
	assert.NoError(t, err)

	// Verify destination contents
	// file1.txt
	content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(content1))

	// nested/file2.txt
	content2, err := os.ReadFile(filepath.Join(dstDir, "nested", "file2.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "world", string(content2))
}

func TestCopyDir_SourceNotExist(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "nonexistent")
	dstDir := filepath.Join(t.TempDir(), "dest_dir")

	err := copyDir(srcDir, dstDir)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err), "Expected a non-exist error")
}

func TestCopyFile(t *testing.T) {
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "source.txt")
	err := os.WriteFile(srcFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	dstDir := t.TempDir()
	dstFile := filepath.Join(dstDir, "destination.txt")

	err = copyFile(srcFile, dstFile)
	assert.NoError(t, err)

	content, err := os.ReadFile(dstFile)
	assert.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestCopyFile_SourceNotExist(t *testing.T) {
	srcFile := filepath.Join(t.TempDir(), "nonexistent.txt")
	dstFile := filepath.Join(t.TempDir(), "destination.txt")

	err := copyFile(srcFile, dstFile)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err), "Expected a non-exist error")
}
