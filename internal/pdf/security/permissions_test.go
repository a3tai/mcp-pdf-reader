package security

import (
	"strings"
	"testing"
)

func TestPermissions_FromInt32(t *testing.T) {
	tests := []struct {
		name  string
		perms int32
		want  Permissions
	}{
		{
			name:  "All permissions granted",
			perms: -1, // All bits set
			want: Permissions{
				Print:            true,
				Modify:           true,
				Copy:             true,
				Annotate:         true,
				FillForms:        true,
				Extract:          true,
				Assemble:         true,
				PrintHighQuality: true,
			},
		},
		{
			name:  "No permissions granted",
			perms: int32(-4096), // 0xFFFFF000 - Only required bits set
			want: Permissions{
				Print:            false,
				Modify:           false,
				Copy:             false,
				Annotate:         false,
				FillForms:        false,
				Extract:          false,
				Assemble:         false,
				PrintHighQuality: true,
			},
		},
		{
			name:  "Print only",
			perms: int32(-4092), // 0xFFFFF004 - Bit 3 set
			want: Permissions{
				Print:            true,
				Modify:           false,
				Copy:             false,
				Annotate:         false,
				FillForms:        false,
				Extract:          false,
				Assemble:         false,
				PrintHighQuality: true,
			},
		},
		{
			name:  "Copy and extract only",
			perms: int32(-3056), // 0xFFFFF410 - Bits 5 and 10 set
			want: Permissions{
				Print:            false,
				Modify:           false,
				Copy:             true,
				Annotate:         false,
				FillForms:        false,
				Extract:          true,
				Assemble:         false,
				PrintHighQuality: true,
			},
		},
		{
			name:  "Typical restricted permissions (-44)",
			perms: -44, // Common value: allows print, copy, forms, extract, assemble, high-quality print
			want: Permissions{
				Print:            true,
				Modify:           false,
				Copy:             true,
				Annotate:         false,
				FillForms:        true,
				Extract:          true,
				Assemble:         true,
				PrintHighQuality: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Permissions{}.FromInt32(tt.perms)

			if got.Print != tt.want.Print {
				t.Errorf("Print = %v, want %v", got.Print, tt.want.Print)
			}
			if got.Modify != tt.want.Modify {
				t.Errorf("Modify = %v, want %v", got.Modify, tt.want.Modify)
			}
			if got.Copy != tt.want.Copy {
				t.Errorf("Copy = %v, want %v", got.Copy, tt.want.Copy)
			}
			if got.Annotate != tt.want.Annotate {
				t.Errorf("Annotate = %v, want %v", got.Annotate, tt.want.Annotate)
			}
			if got.FillForms != tt.want.FillForms {
				t.Errorf("FillForms = %v, want %v", got.FillForms, tt.want.FillForms)
			}
			if got.Extract != tt.want.Extract {
				t.Errorf("Extract = %v, want %v", got.Extract, tt.want.Extract)
			}
			if got.Assemble != tt.want.Assemble {
				t.Errorf("Assemble = %v, want %v", got.Assemble, tt.want.Assemble)
			}
			if got.PrintHighQuality != tt.want.PrintHighQuality {
				t.Errorf("PrintHighQuality = %v, want %v", got.PrintHighQuality, tt.want.PrintHighQuality)
			}
		})
	}
}

func TestPermissions_ToInt32(t *testing.T) {
	tests := []struct {
		name  string
		perms Permissions
	}{
		{
			name: "All permissions granted",
			perms: Permissions{
				Print:            true,
				Modify:           true,
				Copy:             true,
				Annotate:         true,
				FillForms:        true,
				Extract:          true,
				Assemble:         true,
				PrintHighQuality: true,
			},
		},
		{
			name: "No permissions granted",
			perms: Permissions{
				Print:            false,
				Modify:           false,
				Copy:             false,
				Annotate:         false,
				FillForms:        false,
				Extract:          false,
				Assemble:         false,
				PrintHighQuality: false,
			},
		},
		{
			name: "Mixed permissions",
			perms: Permissions{
				Print:            true,
				Modify:           false,
				Copy:             true,
				Annotate:         false,
				FillForms:        true,
				Extract:          false,
				Assemble:         true,
				PrintHighQuality: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to int32 and back
			int32Val := tt.perms.ToInt32()
			roundTrip := Permissions{}.FromInt32(int32Val)

			// Should be identical after round trip
			if roundTrip != tt.perms {
				t.Errorf("Round trip failed: got %+v, want %+v", roundTrip, tt.perms)
			}

			// Check that required bits are set
			// Bits 1, 2, 7, 8 should always be set to 1
			requiredBits := int32Val & 0xC3
			expectedRequired := int32(0xC3)
			if requiredBits != expectedRequired {
				t.Errorf("Required bits not set correctly: got %x, want %x", requiredBits, expectedRequired)
			}

			// Check that high bits (13-32) are set for encrypted documents (but not bit 12)
			highBits := int32Val & int32(-8192) // 0xFFFFE000
			expectedHigh := int32(-8192)        // 0xFFFFE000
			if highBits != expectedHigh {
				t.Errorf("High bits not set correctly: got %x, want %x", highBits, expectedHigh)
			}
		})
	}
}

func TestPermissions_HasPermission(t *testing.T) {
	perms := Permissions{
		Print:            true,
		Modify:           false,
		Copy:             true,
		Annotate:         false,
		FillForms:        true,
		Extract:          false,
		Assemble:         true,
		PrintHighQuality: false,
	}

	tests := []struct {
		permission string
		want       bool
	}{
		{"print", true},
		{"Print", true},
		{"PRINT", true},
		{"modify", false},
		{"copy", true},
		{"annotate", false},
		{"fillforms", true},
		{"fill_forms", true},
		{"extract", false},
		{"assemble", true},
		{"printhighquality", false},
		{"print_high_quality", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.permission, func(t *testing.T) {
			got := perms.HasPermission(tt.permission)
			if got != tt.want {
				t.Errorf("HasPermission(%q) = %v, want %v", tt.permission, got, tt.want)
			}
		})
	}
}

func TestPermissions_IsRestricted(t *testing.T) {
	tests := []struct {
		name  string
		perms Permissions
		want  bool
	}{
		{
			name: "All permissions granted",
			perms: Permissions{
				Print:            true,
				Modify:           true,
				Copy:             true,
				Annotate:         true,
				FillForms:        true,
				Extract:          true,
				Assemble:         true,
				PrintHighQuality: true,
			},
			want: false,
		},
		{
			name: "No permissions granted",
			perms: Permissions{
				Print:            false,
				Modify:           false,
				Copy:             false,
				Annotate:         false,
				FillForms:        false,
				Extract:          false,
				Assemble:         false,
				PrintHighQuality: false,
			},
			want: true,
		},
		{
			name: "One permission denied",
			perms: Permissions{
				Print:            false, // This one denied
				Modify:           true,
				Copy:             true,
				Annotate:         true,
				FillForms:        true,
				Extract:          true,
				Assemble:         true,
				PrintHighQuality: true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.perms.IsRestricted()
			if got != tt.want {
				t.Errorf("IsRestricted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPermissions_GetAllowedOperations(t *testing.T) {
	perms := Permissions{
		Print:            true,
		Modify:           false,
		Copy:             true,
		Annotate:         false,
		FillForms:        true,
		Extract:          false,
		Assemble:         true,
		PrintHighQuality: false,
	}

	allowed := perms.GetAllowedOperations()
	expected := []string{"print", "copy", "fill_forms", "assemble"}

	if len(allowed) != len(expected) {
		t.Errorf("GetAllowedOperations() length = %d, want %d", len(allowed), len(expected))
	}

	for _, exp := range expected {
		found := false
		for _, actual := range allowed {
			if actual == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected operation %q not found in allowed operations", exp)
		}
	}
}

func TestPermissions_GetDeniedOperations(t *testing.T) {
	perms := Permissions{
		Print:            true,
		Modify:           false,
		Copy:             true,
		Annotate:         false,
		FillForms:        true,
		Extract:          false,
		Assemble:         true,
		PrintHighQuality: false,
	}

	denied := perms.GetDeniedOperations()
	expected := []string{"modify", "annotate", "extract", "print_high_quality"}

	if len(denied) != len(expected) {
		t.Errorf("GetDeniedOperations() length = %d, want %d", len(denied), len(expected))
	}

	for _, exp := range expected {
		found := false
		for _, actual := range denied {
			if actual == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected operation %q not found in denied operations", exp)
		}
	}
}

func TestPermissions_String(t *testing.T) {
	tests := []struct {
		name     string
		perms    Permissions
		contains []string
	}{
		{
			name: "All permissions granted",
			perms: Permissions{
				Print:            true,
				Modify:           true,
				Copy:             true,
				Annotate:         true,
				FillForms:        true,
				Extract:          true,
				Assemble:         true,
				PrintHighQuality: true,
			},
			contains: []string{"Allowed:", "Print", "Modify", "Copy", "Annotate", "FillForms", "Extract", "Assemble", "PrintHighQuality"},
		},
		{
			name: "No permissions granted",
			perms: Permissions{
				Print:            false,
				Modify:           false,
				Copy:             false,
				Annotate:         false,
				FillForms:        false,
				Extract:          false,
				Assemble:         false,
				PrintHighQuality: false,
			},
			contains: []string{"No permissions granted"},
		},
		{
			name: "Some permissions granted",
			perms: Permissions{
				Print:            true,
				Modify:           false,
				Copy:             true,
				Annotate:         false,
				FillForms:        false,
				Extract:          false,
				Assemble:         false,
				PrintHighQuality: false,
			},
			contains: []string{"Allowed:", "Print", "Copy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.perms.String()

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("String() result should contain %q, got: %q", expected, result)
				}
			}

			// Make sure we don't have contradictory content
			if strings.Contains(result, "No permissions granted") && strings.Contains(result, "Allowed:") {
				t.Errorf("String() should not contain both 'No permissions granted' and 'Allowed:', got: %q", result)
			}
		})
	}
}

func TestNewPermissions(t *testing.T) {
	testPerms := int32(-44)
	perms := NewPermissions(testPerms)

	// Should be equivalent to calling FromInt32
	expected := Permissions{}.FromInt32(testPerms)

	if perms != expected {
		t.Errorf("NewPermissions() = %+v, want %+v", perms, expected)
	}
}

func TestNewFullPermissions(t *testing.T) {
	perms := NewFullPermissions()

	if !perms.Print || !perms.Modify || !perms.Copy || !perms.Annotate ||
		!perms.FillForms || !perms.Extract || !perms.Assemble || !perms.PrintHighQuality {
		t.Errorf("NewFullPermissions() should grant all permissions, got %+v", perms)
	}

	if perms.IsRestricted() {
		t.Error("NewFullPermissions() result should not be restricted")
	}
}

func TestNewNoPermissions(t *testing.T) {
	perms := NewNoPermissions()

	if perms.Print || perms.Modify || perms.Copy || perms.Annotate ||
		perms.FillForms || perms.Extract || perms.Assemble || perms.PrintHighQuality {
		t.Errorf("NewNoPermissions() should deny all permissions, got %+v", perms)
	}

	if !perms.IsRestricted() {
		t.Error("NewNoPermissions() result should be restricted")
	}
}

// Test specific bit patterns from real PDF files
func TestPermissions_RealWorldValues(t *testing.T) {
	tests := []struct {
		name        string
		value       int32
		description string
	}{
		{
			name:        "Adobe Acrobat default restricted",
			value:       -44,
			description: "Typical Adobe Acrobat restriction: no print, no modify",
		},
		{
			name:        "No restrictions",
			value:       -4,
			description: "All permissions granted",
		},
		{
			name:        "Print and copy only",
			value:       -60,
			description: "Allow print and copy, restrict everything else",
		},
		{
			name:        "Form filling only",
			value:       -524,
			description: "Allow form filling only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := NewPermissions(tt.value)

			// Verify permissions parsing works correctly
			if perms.String() == "" {
				t.Error("Permissions parsing should not result in empty string")
			}

			// Verify that conversion to int32 produces a valid permissions value
			// (don't require exact round-trip due to PDF generator differences in reserved bits)
			converted := perms.ToInt32()
			backConverted := NewPermissions(converted)
			if backConverted != perms {
				t.Errorf("Conversion consistency failed: original perms %+v != back-converted %+v", perms, backConverted)
			}

			t.Logf("%s (%d): %s", tt.name, tt.value, perms.String())
		})
	}
}

// Benchmark tests
func BenchmarkPermissions_FromInt32(b *testing.B) {
	testVal := int32(-44)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Permissions{}.FromInt32(testVal)
	}
}

func BenchmarkPermissions_ToInt32(b *testing.B) {
	perms := NewFullPermissions()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		perms.ToInt32()
	}
}

func BenchmarkPermissions_HasPermission(b *testing.B) {
	perms := NewFullPermissions()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		perms.HasPermission("print")
	}
}

func BenchmarkPermissions_String(b *testing.B) {
	perms := NewFullPermissions()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		perms.String()
	}
}

// Test edge cases and error conditions
func TestPermissions_EdgeCases(t *testing.T) {
	t.Run("Zero value", func(t *testing.T) {
		perms := NewPermissions(0)
		if !perms.IsRestricted() {
			t.Error("Zero permissions should be restricted")
		}
	})

	t.Run("Maximum int32", func(t *testing.T) {
		perms := NewPermissions(2147483647) // Max int32
		// Should still work without panicking
		_ = perms.String()
		_ = perms.IsRestricted()
	})

	t.Run("Minimum int32", func(t *testing.T) {
		perms := NewPermissions(-2147483648) // Min int32
		// Should still work without panicking
		_ = perms.String()
		_ = perms.IsRestricted()
	})

	t.Run("HasPermission with various cases", func(t *testing.T) {
		perms := NewFullPermissions()

		// Test case variations
		testCases := []string{
			"print", "Print", "PRINT",
			"fill_forms", "fillforms", "FillForms", "FILL_FORMS",
			"print_high_quality", "printhighquality", "PrintHighQuality",
		}

		for _, testCase := range testCases {
			// Should not panic
			_ = perms.HasPermission(testCase)
		}
	})
}
