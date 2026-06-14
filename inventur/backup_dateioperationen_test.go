package inventur

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyFile(t *testing.T) {
	t.Run("successful copy", func(t *testing.T) {
		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "src.txt")
		dst := filepath.Join(tempDir, "dst.txt")

		err := os.WriteFile(src, []byte("hello world"), 0644)
		assert.NoError(t, err)

		err = copyFile(src, dst)
		assert.NoError(t, err)

		content, err := os.ReadFile(dst)
		assert.NoError(t, err)
		assert.Equal(t, []byte("hello world"), content)
	})

	t.Run("source file does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "nonexistent.txt")
		dst := filepath.Join(tempDir, "dst.txt")

		err := copyFile(src, dst)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("destination cannot be created", func(t *testing.T) {
		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "src.txt")
		err := os.WriteFile(src, []byte("hello world"), 0644)
		assert.NoError(t, err)

		// Create a directory where the destination file should be
		dstDir := filepath.Join(tempDir, "dst")
		err = os.Mkdir(dstDir, 0755)
		assert.NoError(t, err)

		// dst is a directory, so os.Create will fail
		err = copyFile(src, dstDir)
		assert.Error(t, err)
	})
}
