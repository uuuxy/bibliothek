package api

import (
	"testing"
	"time"
)

func TestCalculateDueDate(t *testing.T) {
	// 1. Test normal book (should be +28 days / 4 weeks)
	dueNormal := calculateDueDate("Some regular book title")
	expectedNormal := time.Now().AddDate(0, 0, 28)
	// Check if the difference is less than a few seconds (to prevent test timing failure)
	if dueNormal.Sub(expectedNormal).Abs() > 2*time.Second {
		t.Errorf("Expected normal due date close to %v, got %v", expectedNormal, dueNormal)
	}

	// 2. Test LMF book (starts with lmf-)
	dueLMF := calculateDueDate("lmf-chemistry-class-10")
	now := time.Now()
	expectedYear := now.Year()
	if now.Month() >= time.August {
		expectedYear++
	}

	if dueLMF.Year() != expectedYear {
		t.Errorf("Expected LMF year to be %d, got %d", expectedYear, dueLMF.Year())
	}
	if dueLMF.Month() != time.July {
		t.Errorf("Expected LMF month to be July, got %v", dueLMF.Month())
	}
	if dueLMF.Day() != 31 {
		t.Errorf("Expected LMF day to be 31, got %d", dueLMF.Day())
	}
	if dueLMF.Hour() != 23 || dueLMF.Minute() != 59 || dueLMF.Second() != 59 {
		t.Errorf("Expected LMF time to be 23:59:59, got %02d:%02d:%02d", dueLMF.Hour(), dueLMF.Minute(), dueLMF.Second())
	}

	// 3. Test case insensitivity
	dueLMFUpper := calculateDueDate("LMF-BIOLOGY")
	if dueLMFUpper.Year() != expectedYear || dueLMFUpper.Month() != time.July || dueLMFUpper.Day() != 31 {
		t.Errorf("Expected case-insensitive matching for 'LMF-BIOLOGY' to yield July 31st %d, got %v", expectedYear, dueLMFUpper)
	}
}
