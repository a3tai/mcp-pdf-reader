package security

import (
	"fmt"
	"strings"
)

// Permissions represents PDF document permissions based on the P entry in the encryption dictionary
type Permissions struct {
	Print            bool // Bit 3 - Print the document
	Modify           bool // Bit 4 - Modify the contents of the document
	Copy             bool // Bit 5 - Copy or extract text and graphics
	Annotate         bool // Bit 6 - Add or modify text annotations, fill in form fields
	FillForms        bool // Bit 9 - Fill in existing interactive form fields (including signature fields)
	Extract          bool // Bit 10 - Extract text and graphics (in support of accessibility)
	Assemble         bool // Bit 11 - Assemble the document (insert, rotate, or delete pages and create bookmarks or thumbnail images)
	PrintHighQuality bool // Bit 12 - Print the document to a representation from which a faithful digital copy could be generated
}

// FromInt32 converts a PDF permissions integer to a Permissions struct
// PDF permissions are stored as a signed 32-bit integer with specific bit flags
func (p Permissions) FromInt32(perms int32) Permissions {
	return Permissions{
		Print:            (perms & 0x04) != 0,   // Bit 3 (2^2 = 4)
		Modify:           (perms & 0x08) != 0,   // Bit 4 (2^3 = 8)
		Copy:             (perms & 0x10) != 0,   // Bit 5 (2^4 = 16)
		Annotate:         (perms & 0x20) != 0,   // Bit 6 (2^5 = 32)
		FillForms:        (perms & 0x200) != 0,  // Bit 9 (2^8 = 512)
		Extract:          (perms & 0x400) != 0,  // Bit 10 (2^9 = 1024)
		Assemble:         (perms & 0x800) != 0,  // Bit 11 (2^10 = 2048)
		PrintHighQuality: (perms & 0x1000) != 0, // Bit 12 (2^11 = 4096)
	}
}

// ToInt32 converts a Permissions struct back to a PDF permissions integer
func (p Permissions) ToInt32() int32 {
	var perms int32

	// Set the required bits (bits 1, 2, 7, 8) - these must always be set to 1
	perms |= 0x03 // Bits 1-2
	perms |= 0xC0 // Bits 7-8

	// Set permission bits based on boolean values
	if p.Print {
		perms |= 0x04 // Bit 3
	}
	if p.Modify {
		perms |= 0x08 // Bit 4
	}
	if p.Copy {
		perms |= 0x10 // Bit 5
	}
	if p.Annotate {
		perms |= 0x20 // Bit 6
	}
	if p.FillForms {
		perms |= 0x200 // Bit 9
	}
	if p.Extract {
		perms |= 0x400 // Bit 10
	}
	if p.Assemble {
		perms |= 0x800 // Bit 11
	}
	if p.PrintHighQuality {
		perms |= 0x1000 // Bit 12
	}

	// For encrypted documents, bits 13-32 should be set to 1 (but not bit 12)
	perms |= int32(-8192) // 0xFFFFE000 as signed int32

	return perms
}

// HasPermission checks if a specific permission is granted
func (p Permissions) HasPermission(permission string) bool {
	switch strings.ToLower(permission) {
	case "print":
		return p.Print
	case "modify":
		return p.Modify
	case "copy":
		return p.Copy
	case "annotate":
		return p.Annotate
	case "fillforms", "fill_forms":
		return p.FillForms
	case "extract":
		return p.Extract
	case "assemble":
		return p.Assemble
	case "printhighquality", "print_high_quality":
		return p.PrintHighQuality
	default:
		return false
	}
}

// IsRestricted returns true if any permissions are denied
func (p Permissions) IsRestricted() bool {
	return !p.Print || !p.Modify || !p.Copy || !p.Annotate ||
		!p.FillForms || !p.Extract || !p.Assemble || !p.PrintHighQuality
}

// GetAllowedOperations returns a list of allowed operations
func (p Permissions) GetAllowedOperations() []string {
	var allowed []string

	if p.Print {
		allowed = append(allowed, "print")
	}
	if p.Modify {
		allowed = append(allowed, "modify")
	}
	if p.Copy {
		allowed = append(allowed, "copy")
	}
	if p.Annotate {
		allowed = append(allowed, "annotate")
	}
	if p.FillForms {
		allowed = append(allowed, "fill_forms")
	}
	if p.Extract {
		allowed = append(allowed, "extract")
	}
	if p.Assemble {
		allowed = append(allowed, "assemble")
	}
	if p.PrintHighQuality {
		allowed = append(allowed, "print_high_quality")
	}

	return allowed
}

// GetDeniedOperations returns a list of denied operations
func (p Permissions) GetDeniedOperations() []string {
	var denied []string

	if !p.Print {
		denied = append(denied, "print")
	}
	if !p.Modify {
		denied = append(denied, "modify")
	}
	if !p.Copy {
		denied = append(denied, "copy")
	}
	if !p.Annotate {
		denied = append(denied, "annotate")
	}
	if !p.FillForms {
		denied = append(denied, "fill_forms")
	}
	if !p.Extract {
		denied = append(denied, "extract")
	}
	if !p.Assemble {
		denied = append(denied, "assemble")
	}
	if !p.PrintHighQuality {
		denied = append(denied, "print_high_quality")
	}

	return denied
}

// String returns a human-readable representation of the permissions
func (p Permissions) String() string {
	var parts []string

	if p.Print {
		parts = append(parts, "Print")
	}
	if p.Modify {
		parts = append(parts, "Modify")
	}
	if p.Copy {
		parts = append(parts, "Copy")
	}
	if p.Annotate {
		parts = append(parts, "Annotate")
	}
	if p.FillForms {
		parts = append(parts, "FillForms")
	}
	if p.Extract {
		parts = append(parts, "Extract")
	}
	if p.Assemble {
		parts = append(parts, "Assemble")
	}
	if p.PrintHighQuality {
		parts = append(parts, "PrintHighQuality")
	}

	if len(parts) == 0 {
		return "No permissions granted"
	}

	return fmt.Sprintf("Allowed: %s", strings.Join(parts, ", "))
}

// NewPermissions creates a new Permissions struct from an int32 value
func NewPermissions(perms int32) Permissions {
	return Permissions{}.FromInt32(perms)
}

// NewFullPermissions creates a Permissions struct with all permissions granted
func NewFullPermissions() Permissions {
	return Permissions{
		Print:            true,
		Modify:           true,
		Copy:             true,
		Annotate:         true,
		FillForms:        true,
		Extract:          true,
		Assemble:         true,
		PrintHighQuality: true,
	}
}

// NewNoPermissions creates a Permissions struct with no permissions granted
func NewNoPermissions() Permissions {
	return Permissions{
		Print:            false,
		Modify:           false,
		Copy:             false,
		Annotate:         false,
		FillForms:        false,
		Extract:          false,
		Assemble:         false,
		PrintHighQuality: false,
	}
}
