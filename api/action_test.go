package api

import (
	"errors"
	"net/http"
	"testing"

	"bibliothek/internal/service"
)

func TestMapServiceErrorToStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "ErrNotFound maps to 404",
			err:      service.ErrNotFound,
			expected: http.StatusNotFound,
		},
		{
			name:     "ErrBlocked maps to 403",
			err:      service.ErrBlocked,
			expected: http.StatusForbidden,
		},
		{
			name:     "ErrInvalidState maps to 400",
			err:      service.ErrInvalidState,
			expected: http.StatusBadRequest,
		},
		{
			name:     "ErrConflict maps to 409",
			err:      service.ErrConflict,
			expected: http.StatusConflict,
		},
		{
			name:     "Unknown error maps to 500",
			err:      errors.New("some unknown error"),
			expected: http.StatusInternalServerError,
		},
		{
			name:     "Wrapped ErrNotFound maps to 404",
			err:      errors.Join(errors.New("wrapped context"), service.ErrNotFound),
			expected: http.StatusNotFound,
		},
		{
			name:     "Nil error maps to 500",
			err:      nil,
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := mapServiceErrorToStatus(tt.err)
			if actual != tt.expected {
				t.Errorf("mapServiceErrorToStatus(%v) = %v; want %v", tt.err, actual, tt.expected)
			}
		})
	}
}
