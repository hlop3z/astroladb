# AstrolaDB IR Examples

**Version**: 1.0.0
**Last Updated**: 2026-02-14

This document provides real-world examples of the AstrolaDB Intermediate Representation (IR).

---

## Table of Contents

1. [Simple Tables](#simple-tables)
2. [Relationships](#relationships)
3. [Validation Constraints](#validation-constraints)
4. [Advanced Features](#advanced-features)
5. [Complete Schema Example](#complete-schema-example)

---

## Simple Tables

### User Table

**JavaScript DSL** (`schemas/auth/user.js`):

```javascript
export default table("user", (t) => {
  t.id();
  t.string("email", 255).unique();
  t.string("username", 50).unique();
  t.string("first_name", 100);
  t.string("last_name", 100);
  t.boolean("is_active").default(true);
  t.timestamps();
});
```

**Resulting IR** (simplified):

```json
{
  "namespace": "auth",
  "name": "user",
  "columns": [
    {
      "name": "id",
      "type": "id",
      "primaryKey": true,
      "nullable": false
    },
    {
      "name": "email",
      "type": "string",
      "typeArgs": [255],
      "unique": true,
      "nullable": false
    },
    {
      "name": "username",
      "type": "string",
      "typeArgs": [50],
      "unique": true,
      "nullable": false
    },
    {
      "name": "first_name",
      "type": "string",
      "typeArgs": [100],
      "nullable": false
    },
    {
      "name": "last_name",
      "type": "string",
      "typeArgs": [100],
      "nullable": false
    },
    {
      "name": "is_active",
      "type": "boolean",
      "default": true,
      "defaultSet": true,
      "nullable": false
    },
    {
      "name": "created_at",
      "type": "datetime",
      "default": { "_type": "sql_expr", "postgres": "NOW()", "sqlite": "CURRENT_TIMESTAMP" },
      "nullable": false
    },
    {
      "name": "updated_at",
      "type": "datetime",
      "default": { "_type": "sql_expr", "postgres": "NOW()", "sqlite": "CURRENT_TIMESTAMP" },
      "nullable": false
    }
  ],
  "indexes": [
    {
      "columns": ["email"],
      "unique": true,
      "name": "auth_user_email_idx"
    },
    {
      "columns": ["username"],
      "unique": true,
      "name": "auth_user_username_idx"
    }
  ]
}
```

**Generated SQL (PostgreSQL)**:

```sql
CREATE TABLE auth_user (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE,
  username VARCHAR(50) NOT NULL UNIQUE,
  first_name VARCHAR(100) NOT NULL,
  last_name VARCHAR(100) NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX auth_user_email_idx ON auth_user(email);
CREATE UNIQUE INDEX auth_user_username_idx ON auth_user(username);
```

---

### Product Table with Enum

**JavaScript DSL** (`schemas/catalog/product.js`):

```javascript
export default table("product", (t) => {
  t.id();
  t.string("name", 200);
  t.text("description").nullable();
  t.decimal("price", 10, 2).min(0);
  t.integer("stock").default(0);
  t.enum("status", ["draft", "active", "archived"]).default("draft");
  t.timestamps();
});
```

**Resulting IR** (simplified):

```json
{
  "namespace": "catalog",
  "name": "product",
  "columns": [
    {
      "name": "id",
      "type": "id",
      "primaryKey": true,
      "nullable": false
    },
    {
      "name": "name",
      "type": "string",
      "typeArgs": [200],
      "nullable": false
    },
    {
      "name": "description",
      "type": "text",
      "nullable": true
    },
    {
      "name": "price",
      "type": "decimal",
      "typeArgs": [10, 2],
      "min": 0,
      "nullable": false
    },
    {
      "name": "stock",
      "type": "integer",
      "default": 0,
      "defaultSet": true,
      "nullable": false
    },
    {
      "name": "status",
      "type": "enum",
      "typeArgs": [["draft", "active", "archived"]],
      "default": "draft",
      "defaultSet": true,
      "nullable": false
    }
  ],
  "checks": [
    {
      "name": "catalog_product_price_check",
      "expression": "price >= 0"
    }
  ]
}
```

**Generated SQL (PostgreSQL)**:

```sql
CREATE TYPE catalog_product_status_enum AS ENUM ('draft', 'active', 'archived');

CREATE TABLE catalog_product (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  name VARCHAR(200) NOT NULL,
  description TEXT,
  price NUMERIC(10,2) NOT NULL,
  stock INTEGER NOT NULL DEFAULT 0,
  status catalog_product_status_enum NOT NULL DEFAULT 'draft',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT catalog_product_price_check CHECK (price >= 0)
);
```

---

## Relationships

### Belongs To (Foreign Key)

**JavaScript DSL** (`schemas/blog/post.js`):

```javascript
export default table("post", (t) => {
  t.id();
  t.belongs_to(".user").onDelete("CASCADE");
  t.string("title", 200);
  t.text("content");
  t.boolean("published").default(false);
  t.timestamps();
});
```

**Resulting IR** (simplified):

```json
{
  "namespace": "blog",
  "name": "post",
  "columns": [
    {
      "name": "id",
      "type": "id",
      "primaryKey": true,
      "nullable": false
    },
    {
      "name": "user_id",
      "type": "uuid",
      "nullable": false,
      "reference": {
        "table": ".user",
        "column": "id",
        "onDelete": "CASCADE",
        "onUpdate": ""
      }
    },
    {
      "name": "title",
      "type": "string",
      "typeArgs": [200],
      "nullable": false
    },
    {
      "name": "content",
      "type": "text",
      "nullable": false
    },
    {
      "name": "published",
      "type": "boolean",
      "default": false,
      "defaultSet": true,
      "nullable": false
    }
  ],
  "foreignKeys": [
    {
      "name": "blog_post_user_id_fkey",
      "columns": ["user_id"],
      "refTable": "auth_user",
      "refColumns": ["id"],
      "onDelete": "CASCADE",
      "onUpdate": ""
    }
  ]
}
```

**Generated SQL (PostgreSQL)**:

```sql
CREATE TABLE blog_post (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  user_id UUID NOT NULL,
  title VARCHAR(200) NOT NULL,
  content TEXT NOT NULL,
  published BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT blog_post_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES auth_user(id) ON DELETE CASCADE
);

CREATE INDEX blog_post_user_id_idx ON blog_post(user_id);
```

---

### Many-to-Many Relationship

**JavaScript DSL** (`schemas/blog/tag.js`):

```javascript
export default table("tag", (t) => {
  t.id();
  t.string("name", 50).unique();
  t.many_to_many("blog.post");
});
```

**Resulting IR** - Auto-generated join table:

```json
{
  "namespace": "blog",
  "name": "post_tag",
  "columns": [
    {
      "name": "post_id",
      "type": "uuid",
      "primaryKey": true,
      "nullable": false,
      "reference": {
        "table": "blog.post",
        "column": "id",
        "onDelete": "CASCADE"
      }
    },
    {
      "name": "tag_id",
      "type": "uuid",
      "primaryKey": true,
      "nullable": false,
      "reference": {
        "table": "blog.tag",
        "column": "id",
        "onDelete": "CASCADE"
      }
    }
  ],
  "foreignKeys": [
    {
      "columns": ["post_id"],
      "refTable": "blog_post",
      "refColumns": ["id"],
      "onDelete": "CASCADE"
    },
    {
      "columns": ["tag_id"],
      "refTable": "blog_tag",
      "refColumns": ["id"],
      "onDelete": "CASCADE"
    }
  ]
}
```

**Generated SQL (PostgreSQL)**:

```sql
CREATE TABLE blog_post_tag (
  post_id UUID NOT NULL,
  tag_id UUID NOT NULL,
  PRIMARY KEY (post_id, tag_id),
  CONSTRAINT blog_post_tag_post_id_fkey FOREIGN KEY (post_id)
    REFERENCES blog_post(id) ON DELETE CASCADE,
  CONSTRAINT blog_post_tag_tag_id_fkey FOREIGN KEY (tag_id)
    REFERENCES blog_tag(id) ON DELETE CASCADE
);

CREATE INDEX blog_post_tag_post_id_idx ON blog_post_tag(post_id);
CREATE INDEX blog_post_tag_tag_id_idx ON blog_post_tag(tag_id);
```

---

## Validation Constraints

### Email Validation

**JavaScript DSL**:

```javascript
export default table("contact", (t) => {
  t.id();
  t.string("email", 255)
    .format("email")
    .pattern("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$");
  t.string("phone", 20).nullable();
});
```

**Resulting IR**:

```json
{
  "columns": [
    {
      "name": "email",
      "type": "string",
      "typeArgs": [255],
      "format": "email",
      "pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
      "nullable": false
    }
  ]
}
```

**Note**: `format` and `pattern` are used by export systems (OpenAPI, TypeScript Zod) and documentation generators. Database CHECK constraints can be added separately.

---

### Range Validation

**JavaScript DSL**:

```javascript
export default table("booking", (t) => {
  t.id();
  t.integer("guests").min(1).max(10);
  t.decimal("amount", 10, 2).min(0);
  t.date("check_in");
  t.date("check_out");
});
```

**Resulting IR**:

```json
{
  "columns": [
    {
      "name": "guests",
      "type": "integer",
      "min": 1,
      "max": 10,
      "nullable": false
    },
    {
      "name": "amount",
      "type": "decimal",
      "typeArgs": [10, 2],
      "min": 0,
      "nullable": false
    }
  ],
  "checks": [
    {
      "name": "booking_guests_check",
      "expression": "guests >= 1 AND guests <= 10"
    },
    {
      "name": "booking_amount_check",
      "expression": "amount >= 0"
    }
  ]
}
```

---

### Custom CHECK Constraints

**JavaScript DSL**:

```javascript
export default table("booking", (t) => {
  t.id();
  t.date("check_in");
  t.date("check_out");
  t.check("check_out > check_in", "check_out_after_check_in");
});
```

**Resulting IR**:

```json
{
  "checks": [
    {
      "name": "check_out_after_check_in",
      "expression": "check_out > check_in"
    }
  ]
}
```

---

## Advanced Features

### Soft Deletes

**JavaScript DSL**:

```javascript
export default table("document", (t) => {
  t.id();
  t.string("title", 200);
  t.text("content");
  t.datetime("deleted_at").nullable();
  t.timestamps();
});
```

**Resulting IR**:

```json
{
  "columns": [
    {
      "name": "deleted_at",
      "type": "datetime",
      "nullable": true
    }
  ]
}
```

**Generated SQL**:

```sql
CREATE TABLE document (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  title VARCHAR(200) NOT NULL,
  content TEXT NOT NULL,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for filtering out soft-deleted records
CREATE INDEX document_deleted_at_idx ON document(deleted_at)
  WHERE deleted_at IS NOT NULL;
```

---

### Partial Index

**JavaScript DSL**:

```javascript
export default table("product", (t) => {
  t.id();
  t.string("sku", 50);
  t.enum("status", ["draft", "active", "archived"]);
  t.index(["sku"], { where: "status = 'active'" });
});
```

**Resulting IR**:

```json
{
  "indexes": [
    {
      "columns": ["sku"],
      "unique": false,
      "where": "status = 'active'",
      "name": "product_sku_idx"
    }
  ]
}
```

**Generated SQL (PostgreSQL)**:

```sql
CREATE INDEX product_sku_idx ON product(sku) WHERE status = 'active';
```

---

### JSON Column

**JavaScript DSL**:

```javascript
export default table("settings", (t) => {
  t.id();
  t.belongs_to(".user").onDelete("CASCADE");
  t.json("preferences").default({
    theme: "light",
    notifications: true,
  });
});
```

**Resulting IR**:

```json
{
  "columns": [
    {
      "name": "preferences",
      "type": "json",
      "default": {
        "theme": "light",
        "notifications": true
      },
      "defaultSet": true,
      "nullable": false
    }
  ]
}
```

**Generated SQL (PostgreSQL)**:

```sql
CREATE TABLE settings (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  user_id UUID NOT NULL,
  preferences JSONB NOT NULL DEFAULT '{"theme":"light","notifications":true}',
  CONSTRAINT settings_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES auth_user(id) ON DELETE CASCADE
);
```

---

### Composite Index

**JavaScript DSL**:

```javascript
export default table("event", (t) => {
  t.id();
  t.string("name", 200);
  t.date("start_date");
  t.date("end_date");
  t.enum("status", ["planned", "active", "completed", "cancelled"]);

  // Composite index for filtering
  t.index(["status", "start_date"]);
});
```

**Resulting IR**:

```json
{
  "indexes": [
    {
      "columns": ["status", "start_date"],
      "unique": false,
      "name": "event_status_start_date_idx"
    }
  ]
}
```

**Generated SQL**:

```sql
CREATE INDEX event_status_start_date_idx ON event(status, start_date);
```

---

## Complete Schema Example

### E-Commerce Schema

**User Table** (`schemas/auth/user.js`):

```javascript
export default table("user", (t) => {
  t.id();
  t.string("email", 255).unique().format("email");
  t.string("username", 50).unique();
  t.string("password_hash", 255);
  t.boolean("email_verified").default(false);
  t.boolean("is_active").default(true);
  t.datetime("last_login").nullable();
  t.timestamps();
});
```

**Address Table** (`schemas/auth/address.js`):

```javascript
export default table("address", (t) => {
  t.id();
  t.belongs_to(".user").onDelete("CASCADE");
  t.string("street", 200);
  t.string("city", 100);
  t.string("state", 50);
  t.string("postal_code", 20);
  t.string("country", 2); // ISO 3166-1 alpha-2
  t.boolean("is_default").default(false);
  t.timestamps();
});
```

**Category Table** (`schemas/catalog/category.js`):

```javascript
export default table("category", (t) => {
  t.id();
  t.string("name", 100).unique();
  t.string("slug", 100).unique();
  t.text("description").nullable();
  t.belongs_to(".category", { as: "parent" }).nullable(); // Self-reference
  t.timestamps();
});
```

**Product Table** (`schemas/catalog/product.js`):

```javascript
export default table("product", (t) => {
  t.id();
  t.belongs_to(".category").onDelete("SET NULL").nullable();
  t.string("name", 200);
  t.string("slug", 200).unique();
  t.text("description").nullable();
  t.decimal("price", 10, 2).min(0);
  t.decimal("compare_at_price", 10, 2).nullable(); // Original price for discounts
  t.integer("stock").default(0).min(0);
  t.enum("status", ["draft", "active", "archived"]).default("draft");
  t.json("metadata").nullable(); // Custom fields
  t.timestamps();

  // Composite index for catalog queries
  t.index(["category_id", "status", "created_at"]);
});
```

**Order Table** (`schemas/order/order.js`):

```javascript
export default table("order", (t) => {
  t.id();
  t.belongs_to("auth.user").onDelete("RESTRICT");
  t.belongs_to("auth.address", { as: "shipping_address" }).onDelete("RESTRICT");
  t.decimal("subtotal", 10, 2).min(0);
  t.decimal("tax", 10, 2).default(0);
  t.decimal("shipping", 10, 2).default(0);
  t.decimal("total", 10, 2).min(0);
  t.enum("status", [
    "pending",
    "processing",
    "shipped",
    "delivered",
    "cancelled",
    "refunded",
  ]).default("pending");
  t.timestamps();

  // CHECK: total = subtotal + tax + shipping
  t.check("total = subtotal + tax + shipping", "order_total_check");

  // Index for user's orders
  t.index(["user_id", "status", "created_at"]);
});
```

**Order Item Table** (`schemas/order/order_item.js`):

```javascript
export default table("order_item", (t) => {
  t.id();
  t.belongs_to(".order").onDelete("CASCADE");
  t.belongs_to("catalog.product").onDelete("RESTRICT");
  t.integer("quantity").min(1);
  t.decimal("unit_price", 10, 2).min(0);
  t.decimal("total_price", 10, 2).min(0);

  // CHECK: total_price = quantity * unit_price
  t.check("total_price = quantity * unit_price", "order_item_total_check");
});
```

**Complete IR Output** (for Product table):

```json
{
  "namespace": "catalog",
  "name": "product",
  "columns": [
    {
      "name": "id",
      "type": "id",
      "primaryKey": true,
      "nullable": false
    },
    {
      "name": "category_id",
      "type": "uuid",
      "nullable": true,
      "reference": {
        "table": ".category",
        "column": "id",
        "onDelete": "SET NULL"
      }
    },
    {
      "name": "name",
      "type": "string",
      "typeArgs": [200],
      "nullable": false
    },
    {
      "name": "slug",
      "type": "string",
      "typeArgs": [200],
      "unique": true,
      "nullable": false
    },
    {
      "name": "description",
      "type": "text",
      "nullable": true
    },
    {
      "name": "price",
      "type": "decimal",
      "typeArgs": [10, 2],
      "min": 0,
      "nullable": false
    },
    {
      "name": "compare_at_price",
      "type": "decimal",
      "typeArgs": [10, 2],
      "nullable": true
    },
    {
      "name": "stock",
      "type": "integer",
      "default": 0,
      "defaultSet": true,
      "min": 0,
      "nullable": false
    },
    {
      "name": "status",
      "type": "enum",
      "typeArgs": [["draft", "active", "archived"]],
      "default": "draft",
      "defaultSet": true,
      "nullable": false
    },
    {
      "name": "metadata",
      "type": "json",
      "nullable": true
    },
    {
      "name": "created_at",
      "type": "datetime",
      "default": { "_type": "sql_expr", "postgres": "NOW()", "sqlite": "CURRENT_TIMESTAMP" },
      "nullable": false
    },
    {
      "name": "updated_at",
      "type": "datetime",
      "default": { "_type": "sql_expr", "postgres": "NOW()", "sqlite": "CURRENT_TIMESTAMP" },
      "nullable": false
    }
  ],
  "indexes": [
    {
      "columns": ["slug"],
      "unique": true,
      "name": "catalog_product_slug_idx"
    },
    {
      "columns": ["category_id", "status", "created_at"],
      "unique": false,
      "name": "catalog_product_category_id_status_created_at_idx"
    }
  ],
  "foreignKeys": [
    {
      "columns": ["category_id"],
      "refTable": "catalog_category",
      "refColumns": ["id"],
      "onDelete": "SET NULL",
      "name": "catalog_product_category_id_fkey"
    }
  ],
  "checks": [
    {
      "name": "catalog_product_price_check",
      "expression": "price >= 0"
    },
    {
      "name": "catalog_product_stock_check",
      "expression": "stock >= 0"
    }
  ]
}
```

---

## Using the IR

### In Code Generators

**TypeScript Model Generator**:

```javascript
export default gen((schema) => {
  const files = {};

  for (const table of schema.tables) {
    const fields = table.columns
      .map((col) => {
        const tsType = mapType(col.type);
        const nullable = col.nullable ? ` | null` : "";
        return `  ${col.name}: ${tsType}${nullable};`;
      })
      .join("\n");

    const model = `
export interface ${pascalCase(table.name)} {
${fields}
}
    `.trim();

    files[`models/${table.name}.ts`] = model;
  }

  return render(files);
});
```

### In Migration Generators

**Diff-based Migration**:

```go
// Compare old schema with new schema
oldSchema := loadSchema("v1")
newSchema := loadSchema("v2")

// Generate operations
ops := diff.Compare(oldSchema, newSchema)

// ops is []ast.Operation with types like:
// - ast.CreateTable
// - ast.AddColumn
// - ast.AlterColumn
// - ast.CreateIndex
// etc.

// Serialize to migration file
for _, op := range ops {
    sql, err := dialect.ToSQL(op)
    // Write to migration file
}
```

---

## Further Reading

- [IR Specification](ir-spec.md) - Complete IR specification
- [Architecture Review](../docs-devs/architecture-review.md) - Platform architecture
- [Generator Type Definitions](../examples/generators/types/generator.d.ts) - TypeScript definitions
- [Example Generators](../examples/generators/generators/) - Real generators (FastAPI, Chi, Axum, tRPC)

---

**Last Updated**: 2026-02-14
**IR Version**: 1.0.0
