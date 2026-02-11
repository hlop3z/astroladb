package astroladb

import (
	"fmt"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/metadata"
)

// TypeConverter interface defines methods for converting Alab types to target language types.
type TypeConverter interface {
	ConvertType(col *ast.ColumnDef) string
	ConvertNullable(baseType string, nullable bool) string
	FormatName(name string) string
}

// BaseConverter provides common functionality for type converters.
type BaseConverter struct {
	TypeMap        map[string]string
	NullableFormat string
}

// ConvertNullable applies the nullable format to a base type.
func (b *BaseConverter) ConvertNullable(baseType string, nullable bool) string {
	if !nullable {
		return baseType
	}
	return fmt.Sprintf(b.NullableFormat, baseType)
}

// exportContext is the internal config used by export functions.
// It combines the public ExportConfig with internal metadata.
type exportContext struct {
	*ExportConfig
	Metadata *metadata.Metadata
}

// typeConverters is a global map of all available type converters.
var typeConverters = map[string]TypeConverter{
	"typescript": NewTypeScriptConverter(),
	"go":         NewGoConverter(),
	"python":     NewPythonConverter(),
	"rust":       NewRustConverter(false),
	"graphql":    NewGraphQLConverter(),
	"openapi":    NewOpenAPIConverter(),
}

// GetConverter returns the type converter for the given format.
func GetConverter(format string) TypeConverter {
	return typeConverters[format]
}
