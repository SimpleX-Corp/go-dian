package dian

import (
	"testing"
)

func TestResponseIsPending(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{"66", true},
		{"00", false},
		{"99", false},
	}

	for _, tt := range tests {
		r := &Response{StatusCode: tt.code}
		if r.IsPending() != tt.expected {
			t.Errorf("IsPending() for %q: got %v, want %v", tt.code, r.IsPending(), tt.expected)
		}
	}
}

func TestResponseIsSuccess(t *testing.T) {
	tests := []struct {
		code    string
		valid   bool
		expected bool
	}{
		{"00", true, true},
		{"00", false, false},
		{"66", true, false},
		{"99", false, false},
	}

	for _, tt := range tests {
		r := &Response{StatusCode: tt.code, IsValid: tt.valid}
		if r.IsSuccess() != tt.expected {
			t.Errorf("IsSuccess() for code=%q valid=%v: got %v, want %v",
				tt.code, tt.valid, r.IsSuccess(), tt.expected)
		}
	}
}

func TestResponseIsError(t *testing.T) {
	tests := []struct {
		code     string
		valid    bool
		expected bool
	}{
		{"99", false, true},
		{"00", false, true},
		{"00", true, false},
		{"66", true, false},
	}

	for _, tt := range tests {
		r := &Response{StatusCode: tt.code, IsValid: tt.valid}
		if r.IsError() != tt.expected {
			t.Errorf("IsError() for code=%q valid=%v: got %v, want %v",
				tt.code, tt.valid, r.IsError(), tt.expected)
		}
	}
}

func TestEnvironmentConstants(t *testing.T) {
	if Habilitacion != "habilitacion" {
		t.Errorf("Habilitacion = %q, want 'habilitacion'", Habilitacion)
	}
	if Produccion != "produccion" {
		t.Errorf("Produccion = %q, want 'produccion'", Produccion)
	}
}
