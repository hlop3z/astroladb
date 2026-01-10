// Package alerr provides standardized error handling for Alab.
package alerr

import (
	"strings"
)

// Semantic type suggestions based on column name patterns
var semanticTypeSuggestions = map[string]string{
	"email":       "t.email(name)",
	"username":    "t.username(name)",
	"password":    "t.password_hash(name)",
	"phone":       "t.phone(name)",
	"mobile":      "t.phone(name)",
	"name":        "t.name(name)",
	"first_name":  "t.name(name)",
	"last_name":   "t.name(name)",
	"title":       "t.title(name)",
	"slug":        "t.slug(name)",
	"url":         "t.url(name)",
	"website":     "t.url(name)",
	"ip":          "t.ip(name)",
	"ip_address":  "t.ip(name)",
	"user_agent":  "t.user_agent(name)",
	"price":       "t.money(name)",
	"amount":      "t.money(name)",
	"cost":        "t.money(name)",
	"total":       "t.money(name)",
	"rate":        "t.percentage(name)",
	"percentage":  "t.percentage(name)",
	"count":       "t.counter(name)",
	"view_count":  "t.counter(name)",
	"quantity":    "t.quantity(name)",
	"body":        "t.body(name)",
	"content":     "t.body(name)",
	"description": "t.summary(name)",
	"summary":     "t.summary(name)",
	"excerpt":     "t.summary(name)",
}

// HasNamespace checks if a reference has a namespace prefix.
func HasNamespace(ref string) bool {
	return strings.Contains(ref, ".")
}

// SuggestSemanticType suggests a semantic type based on column name.
// Returns empty string if no suggestion is available.
func SuggestSemanticType(colName string) string {
	lower := strings.ToLower(colName)

	// Direct match
	if suggestion, ok := semanticTypeSuggestions[lower]; ok {
		return suggestion
	}

	// Partial match - check if name contains a known pattern
	for pattern, suggestion := range semanticTypeSuggestions {
		if strings.Contains(lower, pattern) {
			return suggestion
		}
	}

	return ""
}

// NewMissingNamespaceError creates an error for missing namespace in reference.
func NewMissingNamespaceError(ref string) *Error {
	return New(ErrMissingNamespace,
		"belongs_to(\""+ref+"\") - missing namespace prefix\n"+
			"       Use \"namespace."+ref+"\" or \".\"+ref+\"\" for same namespace",
	).With("reference", ref)
}

// NewMissingLengthError creates an error for string column missing length.
func NewMissingLengthError(colName string) *Error {
	suggestion := SuggestSemanticType(colName)

	if suggestion != "" {
		return New(ErrMissingLength,
			"t.string(\""+colName+"\") - length required\n"+
				"       Use "+suggestion+" or t.string(\""+colName+"\", length)",
		).With("column", colName).With("suggestion", suggestion)
	}

	return New(ErrMissingLength,
		"t.string(\""+colName+"\") - length required\n"+
			"       Use t.string(\""+colName+"\", length)",
	).With("column", colName)
}
