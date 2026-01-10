# OpenAPI Edge Cases Example

This example demonstrates the new chained API and semantic types for schema definitions.

## Key Features Demonstrated

### Chained Relationship API
```js
// Chained methods for clear, readable FK definitions
t.belongs_to("auth.user").as("author")
t.belongs_to("auth.user").as("editor").optional()
t.belongs_to("content.category").as("parent").optional().on_delete("cascade")
t.one_to_one("auth.user")
```

### Semantic Types
```js
t.email("email").unique()           // string(255) + email format + RFC pattern
t.username("username").unique()     // string(50) + pattern + min(3)
t.password_hash("password")         // string(255) + hidden from OpenAPI
t.flag("is_active", true)           // boolean + default(true)
t.flag("is_verified")               // boolean + default(false)
t.counter("view_count")             // integer + default(0)
t.slug("slug")                      // string(255) + unique + slug pattern
t.url("website").optional()         // string(2048) + uri format
t.name("display_name")              // string(100)
t.title("title")                    // string(200)
t.body("content")                   // text
t.summary("excerpt").optional()     // string(500)
t.money("price")                    // decimal(19,4) + min(0)
t.quantity("stock")                 // integer + min(0)
```

### Cleaner Unique Constraints
```js
// Simple composite uniqueness
t.unique("follower", "following")   // Auto-appends _id for relationships
```

## Edge Cases Covered

- Self-referential relationships (user referrals, category parents, comment replies)
- Polymorphic associations (reactions, bookmarks, media attachments)
- Many-to-many relationships (users ↔ roles, posts ↔ tags)
- One-to-one relationships (user → profile)
- Multiple FKs to same table (post author vs editor)

## Schemas

### auth namespace
- `auth.user` - Users with self-referential referral system
- `auth.role` - Roles with JSON permissions
- `auth.profile` - One-to-one profile for users

### content namespace
- `content.category` - Self-referential categories (parent/child)
- `content.post` - Posts with multiple author FKs
- `content.comment` - Comments with reply threading
- `content.tag` - Tags for many-to-many with posts

### social namespace
- `social.follow` - Self-referential follower/following
- `social.reaction` - Polymorphic reactions (like/love on any content)
- `social.bookmark` - Polymorphic bookmarks
- `social.media` - Polymorphic media attachments

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
  "ContentPost": {
    "type": "object",
    "x-table": "content_post",
    "x-namespace": "content",
    "properties": {
      "author_id": {
        "type": "string",
        "format": "uuid",
        "x-ref": "auth.user",
        "x-fk": "auth.user.id"
      },
      "editor_id": {
        "type": "string",
        "format": "uuid",
        "x-ref": "auth.user",
        "x-fk": "auth.user.id"
      }
    }
  }
}
```

This metadata enables code generators and API clients to understand relationships and generate correct associations.
