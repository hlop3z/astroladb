// Package dsl provides fluent builders for defining database schemas and migrations.
// The DSL is designed to be simple, boring, deterministic, and JS-friendly.
package dsl

// FormatDef represents a standard format with its OpenAPI format, regex pattern, and RFC reference.
// Used with ColumnBuilder.Format() to apply both format metadata and validation pattern in one call.
type FormatDef struct {
	// Format is the OpenAPI format hint (e.g., "email", "uri", "uuid")
	Format string
	// Pattern is the RFC-compliant regex pattern for validation
	Pattern string
	// RFC is the reference standard (for documentation)
	RFC string
}

// Standard format definitions with RFC-compliant patterns.
// Use these with ColumnBuilder.Format() for automatic format + pattern application.
var (
	// FormatEmail validates email addresses per RFC 5322 (simplified pattern).
	FormatEmail = FormatDef{
		Format:  "email",
		Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		RFC:     "RFC 5322",
	}

	// FormatURI validates URIs per RFC 3986 (simplified HTTP/HTTPS pattern).
	FormatURI = FormatDef{
		Format:  "uri",
		Pattern: `^https?://[^\s/$.?#].[^\s]*$`,
		RFC:     "RFC 3986",
	}

	// FormatUUID validates UUIDs per RFC 4122 (lowercase hex with dashes).
	FormatUUID = FormatDef{
		Format:  "uuid",
		Pattern: `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
		RFC:     "RFC 4122",
	}

	// FormatDate validates ISO 8601 dates (YYYY-MM-DD).
	FormatDate = FormatDef{
		Format:  "date",
		Pattern: `^\d{4}-\d{2}-\d{2}$`,
		RFC:     "RFC 3339 / ISO 8601",
	}

	// FormatTime validates times per RFC 3339 (HH:MM:SS).
	FormatTime = FormatDef{
		Format:  "time",
		Pattern: `^\d{2}:\d{2}:\d{2}$`,
		RFC:     "RFC 3339",
	}

	// FormatDateTime validates timestamps per RFC 3339 (ISO 8601 with timezone).
	FormatDateTime = FormatDef{
		Format:  "date-time",
		Pattern: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`,
		RFC:     "RFC 3339 / ISO 8601",
	}

	// FormatHostname validates hostnames per RFC 1123.
	FormatHostname = FormatDef{
		Format:  "hostname",
		Pattern: `^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`,
		RFC:     "RFC 1123",
	}

	// FormatIPv4 validates IPv4 addresses per RFC 791.
	FormatIPv4 = FormatDef{
		Format:  "ipv4",
		Pattern: `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`,
		RFC:     "RFC 791",
	}

	// FormatIPv6 validates IPv6 addresses per RFC 4291 (simplified pattern).
	FormatIPv6 = FormatDef{
		Format:  "ipv6",
		Pattern: `^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`,
		RFC:     "RFC 4291",
	}

	// FormatPassword is a UI hint to obscure input (no pattern validation).
	FormatPassword = FormatDef{
		Format:  "password",
		Pattern: "", // No pattern - just UI hint to obscure
		RFC:     "OpenAPI UI hint",
	}
)

// Formats is a map of all standard formats by name.
// Used by JS runtime to expose formats as fmt.email, fmt.uri, etc.
var Formats = map[string]FormatDef{
	"email":     FormatEmail,
	"uri":       FormatURI,
	"uuid":      FormatUUID,
	"date":      FormatDate,
	"time":      FormatTime,
	"datetime": FormatDateTime,
	"hostname":  FormatHostname,
	"ipv4":      FormatIPv4,
	"ipv6":      FormatIPv6,
	"password":  FormatPassword,
}

// GetFormat returns the FormatDef for the given format name, or nil if not found.
func GetFormat(name string) *FormatDef {
	if f, ok := Formats[name]; ok {
		return &f
	}
	return nil
}
