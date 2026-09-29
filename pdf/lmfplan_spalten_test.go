package pdf

import "testing"

func TestLmfHoehe(t *testing.T) {
	tests := []struct {
		name string
		bs   []lmfBaustein
		want int
	}{
		{
			name: "empty",
			bs:   nil,
			want: 0,
		},
		{
			name: "single",
			bs: []lmfBaustein{
				{zeilen: 5},
			},
			want: 5,
		},
		{
			name: "multiple",
			bs: []lmfBaustein{
				{zeilen: 3},
				{zeilen: 2},
				{zeilen: 4},
			},
			want: 9,
		},
		{
			name: "zero and negative",
			bs: []lmfBaustein{
				{zeilen: 0},
				{zeilen: -2},
				{zeilen: 5},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lmfHoehe(tt.bs); got != tt.want {
				t.Errorf("lmfHoehe() = %v, want %v", got, tt.want)
			}
		})
	}
}
