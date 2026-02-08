const SIDEBAR = [
    // ==================== GUIDE ====================
    {
        label: "AstrolaDB (Alab)",
        items: [
            { label: "Introduction", slug: "index" },
            { label: "Quick Start", slug: "quick-start" },
            { label: "Comparison", slug: "comparison" },
            { label: "Exports", slug: "exports" },
            { label: "Commands", slug: "commands" },
            { label: "Tables", slug: "tables" },
            {
                label: "Columns",
                items: [
                    { label: "Low Level", slug: "cols/low-level" },
                    { label: "Semantics", slug: "cols/semantics" },
                    { label: "Relationships", slug: "cols/relationships" },
                    { label: "Computed", slug: "cols/computed" },
                ],
            },
            {
                label: "Tutorial",
                items: [
                    { label: "Your First Project", slug: "tutorial/first-project" },
                    { label: "Adding Relationships", slug: "tutorial/relationships" },
                    { label: "Generating Code", slug: "tutorial/generating-code" },
                ],
            },
            {
                label: "Migrations",
                items: [
                    { label: "Overview", slug: "migrations/overview" },
                    { label: "File Anatomy & Safety", slug: "migrations/guide" },
                    { label: "DSL Reference", slug: "migrations/dsl-reference" },
                    { label: "Squash & Internals", slug: "migrations/advanced" },
                ],
            },
            {
                label: "Advanced Users",
                items: [
                    { label: "Overview", slug: "advanced_users/overview" },
                    { label: "Design Philosophy", slug: "advanced_users/philosophy" },
                    { label: "Why & When: Generators", slug: "advanced_users/generators-intro" },
                    { label: "Reference", slug: "advanced_users/reference" },
                    { label: "Usage", slug: "advanced_users/usage" },
                    { label: "Code Generators", slug: "advanced_users/generators" },
                ],
            },
        ],
    },
];

export default SIDEBAR;
