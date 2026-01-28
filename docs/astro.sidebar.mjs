const SIDEBAR = [
  // ==================== GUIDE ====================
  {
    label: "AstrolaDB (Alab)",
    items: [
      { label: "Introduction", slug: "index" },
      { label: "Comparison", slug: "comparison" },
      { label: "Quick Start", slug: "quick-start" },
      { label: "Commands", slug: "commands" },
      { label: "Tables", slug: "tables" },
      {
        label: "Columns",
        items: [
          { label: "Low Level", slug: "cols/low-level" },
          { label: "Semantics", slug: "cols/semantics" },
          { label: "Relationships", slug: "cols/relationships" },
        ],
      },
      /*
            {
                label: "Getting Started",
                items: [
                    { label: "Introduction", slug: "quick-start" },
                ],
            },
            {
                label: "Core Concepts",
                items: [
                    { label: "How It Works", slug: "concepts/how-it-works" },
                ],
            },
            */
    ],
  },
];

/*
    // ==================== SCHEMA ====================
    {
        label: "Schema",
        collapsed: true,
        items: [
            { label: "Tables", slug: "schema/tables" },
            {
                label: "Columns",
                items: [
                    { label: "Column Basics", slug: "schema/columns/basics" },
                ],
            },
            {
                label: "Semantic Types",
                items: [
                    { label: "Identity", slug: "schema/types/identity" },
                ],
            },
        ],
    },
    // ==================== REFERENCE ====================
    {
        label: "Reference",
        collapsed: true,
        items: [
            {
                label: "CLI",
                items: [
                    { label: "Commands", slug: "cli/commands" },
                ],
            },
            {
                label: "Exports",
                items: [
                    { label: "Overview", slug: "exports/overview" },
                ],
            },
            {
                label: "Live Server",
                items: [
                    { label: "Documentation Server", slug: "server/documentation-server" },
                ],
            },
        ],
    },
    // ==================== EXAMPLES ====================
    {
        label: "Examples",
        collapsed: true,
        items: [
            { label: "Blog", slug: "examples/blog" },
            { label: "E-commerce", slug: "examples/ecommerce" },
        ],
    },
*/

export default SIDEBAR;
