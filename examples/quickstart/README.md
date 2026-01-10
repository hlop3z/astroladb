# Quickstart Example

This example demonstrates IDE autocomplete for Alab schema and migration files.

## Setup

This folder was created with `alab init` which generates:

```
quickstart/
  schemas/          # Table definitions
  migrations/       # Migration files
  types/            # TypeScript definitions (auto-generated)
  alab.yaml         # Configuration
  jsconfig.json     # IDE config for autocomplete
```

## Try It Out

1. Open any `.js` file in `schemas/` or `migrations/`
2. Type `t.` inside a table definition - you'll see all available methods
3. After calling a method like `t.string("name", 100)`, type `.` to see modifiers

## Example Files

- `schemas/auth/user.js` - User table with semantic types
- `schemas/blog/post.js` - Blog post with relationships
- `schemas/blog/comment.js` - Comments with self-referential FK
- `migrations/001_initial.js` - Example migration file

## Semantic Types

Instead of low-level types + format modifiers, use semantic types:

```javascript
// Preferred - semantic types with formats baked in
t.email("email")       // string(255) + email format + RFC pattern
t.username("username") // string(50) + pattern + min(3)
t.password_hash("pw")  // string(255) + hidden from OpenAPI
t.url("website")       // string(2048) + uri format
t.money("price")       // decimal(19,4) + min(0)
t.flag("is_active")    // boolean + default(false)

// Low-level types for custom cases
t.string("custom", 100)
t.decimal("custom_amount", 10, 2)
```

## Regenerating Types

If you accidentally edit the types, regenerate them:

```bash
alab types
```
