// TypeScript tRPC Code Generator
// Generates Zod schemas and tRPC routers from your alab schema.
//
// Usage:
//   alab gen run generators/typescript-trpc -o ./generated

export default gen((schema) => {
  const files = {};

  // ─── Helpers ───────────────────────────────────────────────

  const TYPE_MAP = {
    uuid: "z.string().uuid()",
    string: "z.string()",
    text: "z.string()",
    integer: "z.number().int()",
    float: "z.number()",
    boolean: "z.boolean()",
    date: "z.date()",
    time: "z.string()", // ISO time string
    datetime: "z.date()",
    decimal: "z.number()",
    json: "z.record(z.any())",
    base64: "z.string()",
    enum: "z.enum",
  };

  const AUTO_FIELDS = new Set(["id", "created_at", "updated_at"]);

  const capitalize = (s) => s.charAt(0).toUpperCase() + s.slice(1);
  const pascalCase = (s) => s.split("_").map(capitalize).join("");
  const camelCase = (s) => {
    const pascal = pascalCase(s);
    return pascal.charAt(0).toLowerCase() + pascal.slice(1);
  };

  const zodType = (col) => {
    if (col.type === "enum" && col.enum) {
      const values = col.enum.map((v) => `"${v}"`).join(", ");
      return `z.enum([${values}])`;
    }
    return TYPE_MAP[col.type] || "z.any()";
  };

  const zodField = (col) => {
    let type = zodType(col);

    // Add optional/nullable
    if (col.nullable) {
      type += ".nullable()";
    }

    // Add default
    if (col.default !== undefined && typeof col.default !== "object") {
      if (typeof col.default === "string") {
        type += `.default("${col.default}")`;
      } else if (typeof col.default === "boolean") {
        type += `.default(${col.default})`;
      } else {
        type += `.default(${col.default})`;
      }
    }

    return `  ${col.name}: ${type},`;
  };

  // ─── Schema File ────────────────────────────────────────────

  const SCHEMA_IMPORTS = [
    'import { z } from "zod";',
    "",
  ].join("\n");

  function schemaFile(tables) {
    let out = SCHEMA_IMPORTS;

    for (const table of tables) {
      const cls = pascalCase(table.name);

      // Base schema (create/update — no id, no timestamps)
      out += `\n// ${table.name} schemas\n`;
      out += `export const ${camelCase(table.name)}BaseSchema = z.object({\n`;

      const fields = table.columns.filter((c) => !AUTO_FIELDS.has(c.name));
      if (fields.length === 0) {
        out += "  // No user-defined fields\n";
      } else {
        for (const col of fields) {
          out += `${zodField(col)}\n`;
        }
      }
      out += `});\n\n`;

      // Full schema (adds id + timestamps)
      out += `export const ${camelCase(table.name)}Schema = ${camelCase(table.name)}BaseSchema.extend({\n`;
      out += `  id: z.string().uuid(),\n`;
      if (table.timestamps) {
        out += `  created_at: z.date(),\n`;
        out += `  updated_at: z.date(),\n`;
      }
      out += `});\n\n`;

      // TypeScript types
      out += `export type ${cls}Base = z.infer<typeof ${camelCase(table.name)}BaseSchema>;\n`;
      out += `export type ${cls} = z.infer<typeof ${camelCase(table.name)}Schema>;\n\n`;
    }

    return out;
  }

  // ─── Router File ───────────────────────────────────────────

  function routerFile(ns, tables) {
    const imports = tables
      .map((t) => {
        const cls = pascalCase(t.name);
        const camel = camelCase(t.name);
        return `  ${cls},\n  ${cls}Base,\n  ${camel}Schema,\n  ${camel}BaseSchema,`;
      })
      .join("\n");

    let out =
      'import { z } from "zod";\n' +
      'import { initTRPC, TRPCError } from "@trpc/server";\n\n' +
      `import {\n${imports}\n} from "./schemas";\n\n` +
      "const t = initTRPC.create();\n\n" +
      "const router = t.router;\n" +
      "const publicProcedure = t.procedure;\n\n";

    for (const table of tables) {
      const cls = pascalCase(table.name);
      const camel = camelCase(table.name);
      const name = table.name;

      out += `// ${name} router\n`;
      out += `export const ${camel}Router = router({\n`;

      // List
      out += `  list: publicProcedure\n`;
      out += `    .query(async () => {\n`;
      out += `      // TODO: implement\n`;
      out += `      throw new TRPCError({ code: "NOT_IMPLEMENTED" });\n`;
      out += `    }),\n\n`;

      // Get by ID
      out += `  get: publicProcedure\n`;
      out += `    .input(z.object({ id: z.string().uuid() }))\n`;
      out += `    .query(async ({ input }) => {\n`;
      out += `      // TODO: implement\n`;
      out += `      throw new TRPCError({ code: "NOT_IMPLEMENTED" });\n`;
      out += `    }),\n\n`;

      // Create
      out += `  create: publicProcedure\n`;
      out += `    .input(${camel}BaseSchema)\n`;
      out += `    .mutation(async ({ input }) => {\n`;
      out += `      // TODO: implement\n`;
      out += `      throw new TRPCError({ code: "NOT_IMPLEMENTED" });\n`;
      out += `    }),\n\n`;

      // Update
      out += `  update: publicProcedure\n`;
      out += `    .input(z.object({ id: z.string().uuid(), data: ${camel}BaseSchema }))\n`;
      out += `    .mutation(async ({ input }) => {\n`;
      out += `      // TODO: implement\n`;
      out += `      throw new TRPCError({ code: "NOT_IMPLEMENTED" });\n`;
      out += `    }),\n\n`;

      // Delete
      out += `  delete: publicProcedure\n`;
      out += `    .input(z.object({ id: z.string().uuid() }))\n`;
      out += `    .mutation(async ({ input }) => {\n`;
      out += `      // TODO: implement\n`;
      out += `      throw new TRPCError({ code: "NOT_IMPLEMENTED" });\n`;
      out += `    }),\n`;

      out += `});\n\n`;
    }

    return out;
  }

  // ─── App Router ────────────────────────────────────────────

  function appRouterFile(namespaces, tables) {
    const imports = [];
    const routers = [];

    for (const ns of namespaces) {
      const nsTables = tables[ns];
      for (const table of nsTables) {
        const camel = camelCase(table.name);
        imports.push(`import { ${camel}Router } from "./${ns}/routers";`);
        routers.push(`  ${camel}: ${camel}Router,`);
      }
    }

    return (
      'import { initTRPC } from "@trpc/server";\n\n' +
      imports.join("\n") + "\n\n" +
      "const t = initTRPC.create();\n\n" +
      "export const appRouter = t.router({\n" +
      routers.join("\n") + "\n" +
      "});\n\n" +
      "export type AppRouter = typeof appRouter;\n"
    );
  }

  // ─── Server File ───────────────────────────────────────────

  function serverFile() {
    return (
      '/**\n' +
      ' * tRPC Server (generated by alab)\n' +
      ' *\n' +
      ' * Run:\n' +
      ' *   npm install @trpc/server @trpc/client zod\n' +
      ' *   npm install -D tsx\n' +
      ' *   npx tsx server.ts\n' +
      ' *\n' +
      ' * Example client usage in README.md\n' +
      ' */\n\n' +
      'import { createHTTPServer } from "@trpc/server/adapters/standalone";\n' +
      'import { appRouter } from "./app-router";\n\n' +
      'const server = createHTTPServer({\n' +
      '  router: appRouter,\n' +
      '  createContext() {\n' +
      '    return {};\n' +
      '  },\n' +
      '});\n\n' +
      'server.listen(3000);\n' +
      'console.log("✓ tRPC server running on http://localhost:3000");\n'
    );
  }

  // ─── README ────────────────────────────────────────────────

  function readmeFile() {
    return (
      '# tRPC API (generated by alab)\n\n' +
      '## Setup\n\n' +
      '```bash\n' +
      'npm install @trpc/server @trpc/client zod\n' +
      'npm install -D tsx typescript @types/node\n' +
      '```\n\n' +
      '## Run Server\n\n' +
      '```bash\n' +
      'npx tsx server.ts\n' +
      '```\n\n' +
      '## Example Client\n\n' +
      '```typescript\n' +
      'import { createTRPCClient, httpBatchLink } from "@trpc/client";\n' +
      'import type { AppRouter } from "./app-router";\n\n' +
      'const client = createTRPCClient<AppRouter>({\n' +
      '  links: [httpBatchLink({ url: "http://localhost:3000" })],\n' +
      '});\n\n' +
      '// Usage\n' +
      'const items = await client.users.list.query();\n' +
      'const user = await client.users.get.query({ id: "uuid" });\n' +
      'const created = await client.users.create.mutate({ name: "Alice" });\n' +
      '```\n'
    );
  }

  // ─── Generate ──────────────────────────────────────────────

  for (const [ns, tables] of Object.entries(schema.models)) {
    files[`${ns}/schemas.ts`] = schemaFile(tables);
    files[`${ns}/routers.ts`] = routerFile(ns, tables);
  }

  files["app-router.ts"] = appRouterFile(
    Object.keys(schema.models),
    schema.models
  );
  files["server.ts"] = serverFile();
  files["README.md"] = readmeFile();

  return render(files);
});
