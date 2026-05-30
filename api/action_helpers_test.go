package api

import (
	"testing"
	"time"
)

func TestCalculateDueDate(t *testing.T) {
	// 1. Test normal book (should be +21 days)
	dueNormal := calculateDueDate("Some regular book title", "", "07-31")
	expectedNormal := time.Now().AddDate(0, 0, 21)
	if dueNormal.Sub(expectedNormal).Abs() > 2*time.Second {
		t.Errorf("Expected normal due date close to %v, got %v", expectedNormal, dueNormal)
	}

	// 2. Test LMF book (starts with lmf-)
	dueLMF := calculateDueDate("lmf-chemistry-class-10", "", "07-31")
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
	dueLMFUpper := calculateDueDate("LMF-BIOLOGY", "", "07-31")
	if dueLMFUpper.Year() != expectedYear || dueLMFUpper.Month() != time.July || dueLMFUpper.Day() != 31 {
		t.Errorf("Expected case-insensitive matching for 'LMF-BIOLOGY' to yield July 31st %d, got %v", expectedYear, dueLMFUpper)
	}

	// 4. Test CD media type (should be +7 days)
	dueCD := calculateDueDate("English Listening CD", "CD", "07-31")
	expectedCD := time.Now().AddDate(0, 0, 7)
	if dueCD.Sub(expectedCD).Abs() > 2*time.Second {
		t.Errorf("Expected CD due date close to %v, got %v", expectedCD, dueCD)
	}

	// 5. Test DVD media type (should be +7 days)
	dueDVD := calculateDueDate("Geschichte Film", "DVD", "07-31")
	if dueDVD.Sub(expectedCD).Abs() > 2*time.Second {
		t.Errorf("Expected DVD due date close to %v, got %v", expectedCD, dueDVD)
	}

	// 6. Test custom lmf_stichtag (e.g. August 15)
	dueCustomLMF := calculateDueDate("lmf-math", "", "08-15")
	if dueCustomLMF.Month() != time.August || dueCustomLMF.Day() != 15 {
		t.Errorf("Expected custom LMF stichtag 08-15 to yield Aug 15, got %v", dueCustomLMF)
	}
}
