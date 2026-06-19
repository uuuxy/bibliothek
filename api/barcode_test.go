package api

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateBarcodePNG(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		isQR        bool
		width       int
		height      int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid Code39",
			content:     "B-10001",
			isQR:        false,
			width:       300,
			height:      100,
			expectError: false,
		},
		{
			name:        "Valid QR Code",
			content:     "https://example.com/books/123",
			isQR:        true,
			width:       200,
			height:      200,
			expectError: false,
		},
		{
			name:        "Invalid Dimensions Code39 (Scaling Error)",
			content:     "B-10001",
			isQR:        false,
			width:       0,
			height:      0,
			expectError: true,
			errorMsg:    "failed to scale barcode",
		},
		{
			name:        "Invalid Dimensions QR Code (Scaling Error)",
			content:     "https://example.com",
			isQR:        true,
			width:       -10,
			height:      -10,
			expectError: true,
			errorMsg:    "failed to scale barcode",
		},
		{
			name:        "Content Too Large for QR Code",
			content:     strings.Repeat("A", 8000), // Exceeds QR capacity
			isQR:        true,
			width:       200,
			height:      200,
			expectError: true,
			errorMsg:    "failed to encode barcode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := GenerateBarcodePNG(tt.content, tt.isQR, tt.width, tt.height)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, output)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, output)

				// Verify output is a valid PNG
				config, err := png.DecodeConfig(bytes.NewReader(output))
				assert.NoError(t, err, "Output should be a valid PNG")
				assert.Equal(t, tt.width, config.Width, "PNG width should match")
				assert.Equal(t, tt.height, config.Height, "PNG height should match")
			}
		})
	}
}
