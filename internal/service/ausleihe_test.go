package service

import (
	"testing"
)

func TestNewAusleihDienst(t *testing.T) {
	// NewAusleihDienst legt die Abfragen nur in Felder ab; nil genügt, um das zu prüfen.
	d := NewAusleihDienst(nil, nil, nil, nil, nil)

	if d.pool != nil || d.leserAbfragen != nil || d.katalogAbfragen != nil || d.ausleihAbfragen != nil ||
		d.protokollAbfragen != nil {
		t.Errorf("NewAusleihDienst(nil, …) = %+v, erwartet nur leere Felder", d)
	}
}
