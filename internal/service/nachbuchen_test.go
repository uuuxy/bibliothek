package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNachbuchService(t *testing.T) {
	svc := NewNachbuchService(nil, nil, nil, nil, nil, nil)

	assert.NotNil(t, svc)

	// Assert it's of the correct type
	defaultSvc, ok := svc.(*defaultLoanService)
	assert.True(t, ok)

	// Assert fields are set correctly
	assert.Nil(t, defaultSvc.pool)
	assert.Nil(t, defaultSvc.studentRepo)
	assert.Nil(t, defaultSvc.bookRepo)
	assert.Nil(t, defaultSvc.userRepo)
	assert.Nil(t, defaultSvc.loanRepo)
	assert.Nil(t, defaultSvc.auditRepo)
}
