# Types

## Lost Precision

| Feature                     | PostgreSQL      | SQLite            |
| --------------------------- | --------------- | ----------------- |
| String length preserved     | ✅ VARCHAR(n)   | ❌ TEXT (lost)    |
| Decimal precision preserved | ✅ NUMERIC(p,s) | ❌ NUMERIC (lost) |
| Type/Subtype preserved      | ✅ Clear types  | ❌ Heuristics     |
| Round-trip safe             | ✅ YES          | ❌ NO             |
