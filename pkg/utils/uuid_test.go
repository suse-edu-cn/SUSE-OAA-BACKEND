package utils

import (
	"testing"

	"github.com/google/uuid"
)

func TestGetUUID(t *testing.T) {
	idStr, err := GetUUID()
	if err != nil {
		t.Fatalf("GetUUID failed: %v", err)
	}
	if len(idStr) != 36 {
		t.Errorf("expected uuid length 36, got %d (%s)", len(idStr), idStr)
	}
	parsed, err := uuid.Parse(idStr)
	if err != nil {
		t.Fatalf("uuid.Parse failed: %v", err)
	}
	if parsed.Version() != 7 {
		t.Errorf("expected UUID version 7, got %v", parsed.Version())
	}
}
