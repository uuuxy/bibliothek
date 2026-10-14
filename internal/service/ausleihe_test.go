package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAusleihDienst(t *testing.T) {
	// Create mock repositories
	// In reality we just need anything that implements the interface. Since NewAusleihDienst
	// just assigns these to fields, we can even pass nil for this simple test.

	svc := NewAusleihDienst(nil, nil, nil, nil, nil)

	assert.NotNil(t, svc)

	// Assert it's of the correct type
	defaultSvc, ok := svc.(*ausleihDienst)
	assert.True(t, ok)

	// Assert fields are set correctly
	assert.Nil(t, defaultSvc.pool)
	assert.Nil(t, defaultSvc.leserAbfragen)
	assert.Nil(t, defaultSvc.katalogAbfragen)
	assert.Nil(t, defaultSvc.ausleihAbfragen)
	assert.Nil(t, defaultSvc.protokollAbfragen)
}
