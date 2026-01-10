// Schema: blog.user
// Blog user with profile information demonstrating semantic types

/*
| Semantic Types              | What They Provide                       |
| --------------------------- | --------------------------------------- |
| `col.email()`               | string(255) + email format + RFC pattern|
| `col.username()`            | string(50) + pattern + min(3)           |
| `col.password_hash()`       | string(255) + hidden from OpenAPI       |
| `col.flag([default])`       | boolean + default(false/true)           |
| `col.counter()`             | integer + default(0)                    |
| `col.slug()`                | string(255) + unique + slug pattern     |
| `col.url()`                 | string(2048) + uri format               |
| `col.money()`               | decimal(19,4) + min(0)                  |

| Low-Level Types             | Why                                     |
| --------------------------- | --------------------------------------- |
| `col.id()` (UUID)           | String, safe                            |
| `col.string(n)`             | Safe                                    |
| `col.text()`                | Safe                                    |
| `col.integer()`             | 32-bit, JS-safe (< 2^53)                |
| `col.float()`               | 32-bit                                  |
| `col.decimal()`             | String-serialized, preserves precision  |
| `col.boolean()`             | Safe                                    |
| `col.date()`, `col.time()`, `col.datetime()` | ISO strings            |
| `col.uuid()`                | String                                  |
| `col.json()`                | Native JS object                        |
| `col.base64()`              | String                                  |
| `col.enum()`                | String                                  |
*/

export default table({
  id: col.id(),
  // Semantic types (preferred)
  username: col.username().unique(),
  email: col.email().unique(),
  display_name: col.name(),
  level: col.counter(),
  balance: col.money(),
  // Low-level types (for custom cases)
  credits: col.decimal(10, 2),
  the_date: col.date(),
  the_time: col.time(),
  the_day: col.datetime(),
  the_object: col.json(),
  the_file: col.base64(),
  bio: col.text().optional(),
}).timestamps()
