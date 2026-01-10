// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import rehypeMermaid from "rehype-mermaid";

const Config = {
  title: "Astroladb",
  description: "Schema-first database migrations. Write once, export everywhere.",
  base: "/astroladb",
  site: "https://hlop3z.github.io",
  repo: "https://github.com/hlop3z/astroladb",
};

// https://astro.build/config
export default defineConfig({
  site: Config.site,
  base: Config.base,
  markdown: {
    rehypePlugins: [
      [
        rehypeMermaid,
        {
          strategy: "img-svg",
          dark: true,
        },
      ],
    ],
    syntaxHighlight: {
      excludeLangs: ["mermaid"],
    },
  },
  integrations: [
    starlight({
      title: Config.title,
      description: Config.description,
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: Config.repo,
        },
      ],
      logo: {
        src: "./src/assets/logo.png",
        replacesTitle: false,
      },
      customCss: ["./src/styles/custom.css"],
      sidebar: [
        // ==================== GUIDE ====================
        {
          label: "Guide",
          items: [
            { label: "Home", slug: "index" },
            {
              label: "Getting Started",
              items: [
                { label: "Introduction", slug: "getting-started/introduction" },
                { label: "Installation", slug: "getting-started/installation" },
                { label: "Quick Start", slug: "getting-started/quick-start" },
                { label: "Your First Schema", slug: "getting-started/first-schema" },
              ],
            },
            {
              label: "Core Concepts",
              items: [
                { label: "How It Works", slug: "concepts/how-it-works" },
                { label: "Project Structure", slug: "concepts/project-structure" },
              ],
            },
          ],
        },
        // ==================== SCHEMA ====================
        {
          label: "Schema",
          collapsed: true,
          items: [
            { label: "Tables", slug: "schema/tables" },
            { label: "Relationships", slug: "schema/relationships" },
            {
              label: "Columns",
              items: [
                { label: "Column Basics", slug: "schema/columns/basics" },
                { label: "Column Modifiers", slug: "schema/columns/modifiers" },
                { label: "Low-Level Types", slug: "schema/columns/low-level" },
              ],
            },
            {
              label: "Semantic Types",
              items: [
                { label: "Identity", slug: "schema/types/identity" },
                { label: "Text", slug: "schema/types/text" },
                { label: "Contact", slug: "schema/types/contact" },
                { label: "Auth", slug: "schema/types/auth" },
                { label: "Numbers", slug: "schema/types/numbers" },
                { label: "Flags", slug: "schema/types/flags" },
                { label: "Locale", slug: "schema/types/locale" },
                { label: "Media", slug: "schema/types/media" },
                { label: "Meta", slug: "schema/types/meta" },
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
                { label: "Configuration", slug: "cli/configuration" },
              ],
            },
            {
              label: "Exports",
              items: [
                { label: "Overview", slug: "exports/overview" },
                { label: "Output Examples", slug: "exports/examples" },
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
      ],
      pagefind: true,
      editLink: {
        baseUrl: `${Config.repo}/edit/main/docs/`,
      },
      lastUpdated: true,
    }),
  ],
});
