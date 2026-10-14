package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAusleihDienst(t *testing.T) {
	// NewAusleihDienst legt die Abfragen nur in Felder ab; nil genügt, um das zu prüfen.

	svc := NewAusleihDienst(nil, nil, nil, nil, nil)

	assert.NotNil(t, svc)

	defaultSvc, ok := svc.(*ausleihDienst)
	assert.True(t, ok)

	assert.Nil(t, defaultSvc.pool)
	assert.Nil(t, defaultSvc.leserAbfragen)
	assert.Nil(t, defaultSvc.katalogAbfragen)
	assert.Nil(t, defaultSvc.ausleihAbfragen)
	assert.Nil(t, defaultSvc.protokollAbfragen)
}
