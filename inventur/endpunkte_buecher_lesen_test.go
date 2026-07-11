package inventur

import (
	"reflect"
	"testing"
)

func TestSortiereBuecherNatuerlich(t *testing.T) {
	tests := []struct {
		name     string
		input    []Book
		expected []Book
	}{
		{
			name:     "Empty slice",
			input:    []Book{},
			expected: []Book{},
		},
		{
			name: "Single element",
			input: []Book{
				{Title: "Book A"},
			},
			expected: []Book{
				{Title: "Book A"},
			},
		},
		{
			name: "Natural number sorting",
			input: []Book{
				{Title: "Teil 10"},
				{Title: "Teil 2"},
				{Title: "Teil 1"},
			},
			expected: []Book{
				{Title: "Teil 1"},
				{Title: "Teil 2"},
				{Title: "Teil 10"},
			},
		},
		{
			name: "Case insensitivity",
			input: []Book{
				{Title: "teil 10"},
				{Title: "Teil 2"},
				{Title: "TEIL 1"},
			},
			expected: []Book{
				{Title: "TEIL 1"},
				{Title: "Teil 2"},
				{Title: "teil 10"},
			},
		},
		{
			name: "SortOrder precedence",
			input: []Book{
				{Title: "Zebra", SortOrder: 1},
				{Title: "Affe", SortOrder: 2},
				{Title: "Bär", SortOrder: 0},
			},
			expected: []Book{
				{Title: "Bär", SortOrder: 0},
				{Title: "Zebra", SortOrder: 1},
				{Title: "Affe", SortOrder: 2},
			},
		},
		{
			name: "Fallback sorting",
			input: []Book{
				{Title: "Band 1 Extra"},
				{Title: "Band 1"},
				{Title: "Band 1 Alpha"},
			},
			expected: []Book{
				{Title: "Band 1"},
				{Title: "Band 1 Alpha"},
				{Title: "Band 1 Extra"},
			},
		},
		{
			name: "Mixed numbers and text",
			input: []Book{
				{Title: "Harry Potter 2"},
				{Title: "Harry Potter 10"},
				{Title: "Harry Potter 1"},
				{Title: "Harry Potter"},
			},
			expected: []Book{
				{Title: "Harry Potter"},
				{Title: "Harry Potter 1"},
				{Title: "Harry Potter 2"},
				{Title: "Harry Potter 10"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputCopy := make([]Book, len(tt.input))
			copy(inputCopy, tt.input)

			sortiereBuecherNatuerlich(inputCopy)

			if !reflect.DeepEqual(inputCopy, tt.expected) {
				t.Errorf("sortiereBuecherNatuerlich() = %v, want %v", inputCopy, tt.expected)
			}
		})
	}
}
