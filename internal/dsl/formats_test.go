package dsl

import (
	"regexp"
	"testing"
)

func TestFormatDefinitions(t *testing.T) {
	tests := []struct {
		name       string
		format     FormatDef
		wantFormat string
		wantRFC    string
		hasPattern bool
	}{
		{
			name:       "email",
			format:     FormatEmail,
			wantFormat: "email",
			wantRFC:    "RFC 5322",
			hasPattern: true,
		},
		{
			name:       "uri",
			format:     FormatURI,
			wantFormat: "uri",
			wantRFC:    "RFC 3986",
			hasPattern: true,
		},
		{
			name:       "uuid",
			format:     FormatUUID,
			wantFormat: "uuid",
			wantRFC:    "RFC 4122",
			hasPattern: true,
		},
		{
			name:       "date",
			format:     FormatDate,
			wantFormat: "date",
			wantRFC:    "RFC 3339 / ISO 8601",
			hasPattern: true,
		},
		{
			name:       "time",
			format:     FormatTime,
			wantFormat: "time",
			wantRFC:    "RFC 3339",
			hasPattern: true,
		},
		{
			name:       "datetime",
			format:     FormatDateTime,
			wantFormat: "date-time",
			wantRFC:    "RFC 3339 / ISO 8601",
			hasPattern: true,
		},
		{
			name:       "hostname",
			format:     FormatHostname,
			wantFormat: "hostname",
			wantRFC:    "RFC 1123",
			hasPattern: true,
		},
		{
			name:       "ipv4",
			format:     FormatIPv4,
			wantFormat: "ipv4",
			wantRFC:    "RFC 791",
			hasPattern: true,
		},
		{
			name:       "ipv6",
			format:     FormatIPv6,
			wantFormat: "ipv6",
			wantRFC:    "RFC 4291",
			hasPattern: true,
		},
		{
			name:       "password",
			format:     FormatPassword,
			wantFormat: "password",
			wantRFC:    "OpenAPI UI hint",
			hasPattern: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.format.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", tt.format.Format, tt.wantFormat)
			}
			if tt.format.RFC != tt.wantRFC {
				t.Errorf("RFC = %q, want %q", tt.format.RFC, tt.wantRFC)
			}
			if tt.hasPattern && tt.format.Pattern == "" {
				t.Error("Expected pattern to be non-empty")
			}
			if !tt.hasPattern && tt.format.Pattern != "" {
				t.Errorf("Expected no pattern, got %q", tt.format.Pattern)
			}
		})
	}
}

func TestGetFormat(t *testing.T) {
	tests := []struct {
		name       string
		formatName string
		want       *FormatDef
	}{
		{"email", "email", &FormatEmail},
		{"uri", "uri", &FormatURI},
		{"uuid", "uuid", &FormatUUID},
		{"date", "date", &FormatDate},
		{"time", "time", &FormatTime},
		{"datetime", "datetime", &FormatDateTime},
		{"hostname", "hostname", &FormatHostname},
		{"ipv4", "ipv4", &FormatIPv4},
		{"ipv6", "ipv6", &FormatIPv6},
		{"password", "password", &FormatPassword},
		{"unknown", "nonexistent", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFormat(tt.formatName)
			if tt.want == nil {
				if got != nil {
					t.Errorf("GetFormat(%q) = %v, want nil", tt.formatName, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("GetFormat(%q) = nil, want non-nil", tt.formatName)
			}
			if got.Format != tt.want.Format {
				t.Errorf("Format = %q, want %q", got.Format, tt.want.Format)
			}
		})
	}
}

func TestFormatsMap(t *testing.T) {
	expectedFormats := []string{
		"email", "uri", "uuid", "date", "time",
		"datetime", "hostname", "ipv4", "ipv6", "password",
	}

	for _, name := range expectedFormats {
		if _, ok := Formats[name]; !ok {
			t.Errorf("Formats map missing %q", name)
		}
	}

	if len(Formats) != len(expectedFormats) {
		t.Errorf("Formats map has %d entries, want %d", len(Formats), len(expectedFormats))
	}
}

func TestPatternsAreValidRegex(t *testing.T) {
	for name, format := range Formats {
		if format.Pattern == "" {
			continue // password has no pattern
		}

		t.Run(name, func(t *testing.T) {
			_, err := regexp.Compile(format.Pattern)
			if err != nil {
				t.Errorf("Pattern for %q is not valid regex: %v", name, err)
			}
		})
	}
}

func TestEmailPattern(t *testing.T) {
	pattern := regexp.MustCompile(FormatEmail.Pattern)

	valid := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.com",
		"user123@example.org",
		"a@b.co",
	}

	invalid := []string{
		"user",
		"@example.com",
		"user@",
		"user@.com",
		"user@@example.com",
	}

	for _, email := range valid {
		if !pattern.MatchString(email) {
			t.Errorf("Email pattern should match %q", email)
		}
	}

	for _, email := range invalid {
		if pattern.MatchString(email) {
			t.Errorf("Email pattern should NOT match %q", email)
		}
	}
}

func TestURIPattern(t *testing.T) {
	pattern := regexp.MustCompile(FormatURI.Pattern)

	valid := []string{
		"http://example.com",
		"https://example.com",
		"https://example.com/path",
		"https://example.com/path?query=1",
		"http://localhost:8080",
	}

	invalid := []string{
		"ftp://example.com",
		"example.com",
		"//example.com",
		"http://",
	}

	for _, uri := range valid {
		if !pattern.MatchString(uri) {
			t.Errorf("URI pattern should match %q", uri)
		}
	}

	for _, uri := range invalid {
		if pattern.MatchString(uri) {
			t.Errorf("URI pattern should NOT match %q", uri)
		}
	}
}

func TestUUIDPattern(t *testing.T) {
	pattern := regexp.MustCompile(FormatUUID.Pattern)

	valid := []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"00000000-0000-0000-0000-000000000000",
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	}

	invalid := []string{
		"123e4567e89b12d3a456426614174000",     // no dashes
		"123E4567-E89B-12D3-A456-426614174000", // uppercase (pattern requires lowercase)
		"123e4567-e89b-12d3-a456",              // too short
		"not-a-uuid",
	}

	for _, uuid := range valid {
		if !pattern.MatchString(uuid) {
			t.Errorf("UUID pattern should match %q", uuid)
		}
	}

	for _, uuid := range invalid {
		if pattern.MatchString(uuid) {
			t.Errorf("UUID pattern should NOT match %q", uuid)
		}
	}
}

func TestDatePattern(t *testing.T) {
	pattern := regexp.MustCompile(FormatDate.Pattern)

	valid := []string{
		"2024-01-01",
		"2024-12-31",
		"1999-06-15",
	}

	invalid := []string{
		"2024/01/01",
		"01-01-2024",
		"2024-1-1",
		"20240101",
	}

	for _, date := range valid {
		if !pattern.MatchString(date) {
			t.Errorf("Date pattern should match %q", date)
		}
	}

	for _, date := range invalid {
		if pattern.MatchString(date) {
			t.Errorf("Date pattern should NOT match %q", date)
		}
	}
}

func TestTimePattern(t *testing.T) {
	pattern := regexp.MustCompile(FormatTime.Pattern)

	valid := []string{
		"12:30:45",
		"00:00:00",
		"23:59:59",
	}

	invalid := []string{
		"12:30",
		"12:30:45.123",
		"1:30:45",
	}

	for _, time := range valid {
		if !pattern.MatchString(time) {
			t.Errorf("Time pattern should match %q", time)
		}
	}

	for _, time := range invalid {
		if pattern.MatchString(time) {
			t.Errorf("Time pattern should NOT match %q", time)
		}
	}
}

func TestDateTimePattern(t *testing.T) {
	pattern := regexp.MustCompile(FormatDateTime.Pattern)

	valid := []string{
		"2024-01-01T12:30:45Z",
		"2024-01-01T12:30:45+00:00",
		"2024-01-01T12:30:45-05:00",
		"2024-01-01T12:30:45.123Z",
		"2024-01-01T12:30:45.123456Z",
	}

	invalid := []string{
		"2024-01-01 12:30:45",
		"2024-01-01T12:30:45",
		"2024-01-01",
	}

	for _, dt := range valid {
		if !pattern.MatchString(dt) {
			t.Errorf("DateTime pattern should match %q", dt)
		}
	}

	for _, dt := range invalid {
		if pattern.MatchString(dt) {
			t.Errorf("DateTime pattern should NOT match %q", dt)
		}
	}
}

func TestHostnamePattern(t *testing.T) {
	pattern := regexp.MustCompile(FormatHostname.Pattern)

	valid := []string{
		"example.com",
		"sub.example.com",
		"localhost",
		"my-server",
		"server1",
	}

	invalid := []string{
		"-example.com",
		"example-.com",
		".example.com",
	}

	for _, hostname := range valid {
		if !pattern.MatchString(hostname) {
			t.Errorf("Hostname pattern should match %q", hostname)
		}
	}

	for _, hostname := range invalid {
		if pattern.MatchString(hostname) {
			t.Errorf("Hostname pattern should NOT match %q", hostname)
		}
	}
}

func TestIPv4Pattern(t *testing.T) {
	pattern := regexp.MustCompile(FormatIPv4.Pattern)

	valid := []string{
		"192.168.1.1",
		"0.0.0.0",
		"255.255.255.255",
		"127.0.0.1",
		"10.0.0.1",
	}

	invalid := []string{
		"256.1.1.1",
		"1.1.1",
		"1.1.1.1.1",
		"192.168.1",
		"abc.def.ghi.jkl",
	}

	for _, ip := range valid {
		if !pattern.MatchString(ip) {
			t.Errorf("IPv4 pattern should match %q", ip)
		}
	}

	for _, ip := range invalid {
		if pattern.MatchString(ip) {
			t.Errorf("IPv4 pattern should NOT match %q", ip)
		}
	}
}

func TestIPv6Pattern(t *testing.T) {
	pattern := regexp.MustCompile(FormatIPv6.Pattern)

	valid := []string{
		"2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		"0000:0000:0000:0000:0000:0000:0000:0001",
		"ABCD:EF01:2345:6789:ABCD:EF01:2345:6789",
	}

	invalid := []string{
		"2001:0db8:85a3::8a2e:0370:7334", // shortened format not supported by simple pattern
		"192.168.1.1",
		"2001:db8",
	}

	for _, ip := range valid {
		if !pattern.MatchString(ip) {
			t.Errorf("IPv6 pattern should match %q", ip)
		}
	}

	for _, ip := range invalid {
		if pattern.MatchString(ip) {
			t.Errorf("IPv6 pattern should NOT match %q", ip)
		}
	}
}

func TestPasswordFormat(t *testing.T) {
	// Password format should have no pattern (it's just a UI hint)
	if FormatPassword.Pattern != "" {
		t.Errorf("Password format should have empty pattern, got %q", FormatPassword.Pattern)
	}
	if FormatPassword.Format != "password" {
		t.Errorf("Password format = %q, want %q", FormatPassword.Format, "password")
	}
}

func TestFormatDefEquality(t *testing.T) {
	// Verify that getting from map returns correct values
	emailFromMap := Formats["email"]
	if emailFromMap.Format != FormatEmail.Format {
		t.Errorf("Map email format = %q, want %q", emailFromMap.Format, FormatEmail.Format)
	}
	if emailFromMap.Pattern != FormatEmail.Pattern {
		t.Errorf("Map email pattern = %q, want %q", emailFromMap.Pattern, FormatEmail.Pattern)
	}
	if emailFromMap.RFC != FormatEmail.RFC {
		t.Errorf("Map email RFC = %q, want %q", emailFromMap.RFC, FormatEmail.RFC)
	}
}
