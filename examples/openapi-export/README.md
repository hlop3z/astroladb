# OpenAPI Export Example

This example demonstrates how to export your Alab schemas to OpenAPI 3.0 format with relationship metadata.

## Schemas

- `blog.user` - Blog users
- `blog.post` - Blog posts with author relationship
- `blog.comment` - Comments with post and author relationships

## Usage

### Export to stdout
```bash
alab schema:export --format openapi
```

### Export to file
```bash
alab schema:export --format openapi --output openapi.json
```

### Other formats
```bash
# JSON Schema
alab schema:export --format jsonschema --output schema.json

# TypeScript types
alab schema:export --format typescript --output types.ts
```

## OpenAPI Extensions

The exported OpenAPI schema includes custom extensions for tooling:

### Schema-level
- `x-table`: SQL table name
- `x-namespace`: Schema namespace

### Property-level (foreign keys)
- `x-ref`: Logical reference (e.g., `"blog.user"`)
- `x-fk`: Full FK path (e.g., `"blog.user.id"`)

## Example Output

```json
{
  "BlogPost": {
    "type": "object",
    "x-table": "blog_post",
    "x-namespace": "blog",
    "properties": {
      "author_id": {
        "type": "string",
        "format": "uuid",
        "x-ref": "blog.user",
        "x-fk": "blog.user.id"
      }
    }
  }
}
```

This metadata enables code generators and API clients to understand relationships and generate correct associations.
