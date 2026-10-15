package service

import (
	"testing"
)

func TestNewAusleihDienst(t *testing.T) {
	// NewAusleihDienst legt die Abfragen nur in Felder ab; nil genügt, um das zu prüfen.
	dienst := NewAusleihDienst(nil, nil, nil, nil, nil)

	d, ok := dienst.(*ausleihDienst)
	if !ok {
		t.Fatalf("NewAusleihDienst() = %T, erwartet *ausleihDienst", dienst)
	}
	if d.pool != nil || d.leserAbfragen != nil || d.katalogAbfragen != nil || d.ausleihAbfragen != nil ||
		d.protokollAbfragen != nil {
		t.Errorf("NewAusleihDienst(nil, …) = %+v, erwartet nur leere Felder", d)
	}
}
