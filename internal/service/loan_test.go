package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSperrGrundFehler(t *testing.T) {
	kernErr := errors.New("base error")
	err := &SperrGrundFehler{
		Kern:  kernErr,
		Grund: "some reason",
	}

	assert.Equal(t, "base error: some reason", err.Error())
	assert.Equal(t, kernErr, err.Unwrap())
	assert.True(t, errors.Is(err, kernErr))
}

func TestNewLoanService(t *testing.T) {
	// Create mock repositories
	// In reality we just need anything that implements the interface. Since NewLoanService
	// just assigns these to fields, we can even pass nil for this simple test.

	svc := NewLoanService(nil, nil, nil, nil, nil)

	assert.NotNil(t, svc)

	// Assert it's of the correct type
	defaultSvc, ok := svc.(*defaultLoanService)
	assert.True(t, ok)

	// Assert fields are set correctly
	assert.Nil(t, defaultSvc.pool)
	assert.Nil(t, defaultSvc.studentRepo)
	assert.Nil(t, defaultSvc.bookRepo)
	assert.Nil(t, defaultSvc.loanRepo)
	assert.Nil(t, defaultSvc.auditRepo)
}
