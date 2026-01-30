// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import rehypeMermaid from "rehype-mermaid";
import SIDEBAR from "./astro.sidebar.mjs";

const Config = {
  title: "AstrolaDB",
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
      favicon: "./src/assets/logo.png",
      logo: {
        src: "./src/assets/logo.png",
        replacesTitle: false,
      },
      head: [
        {
          tag: "meta",
          attrs: {
            property: "og:image",
            content: `${Config.site}${Config.base}/og-image.png`,
          },
        },
        {
          tag: "meta",
          attrs: { property: "og:type", content: "website" },
        },
        {
          tag: "meta",
          attrs: { property: "og:site_name", content: Config.title },
        },
        {
          tag: "meta",
          attrs: { name: "twitter:card", content: "summary_large_image" },
        },
      ],
      customCss: ["./src/styles/custom.css"],
      sidebar: SIDEBAR,
      pagefind: true,
      editLink: {
        baseUrl: `${Config.repo}/edit/main/docs/`,
      },
      lastUpdated: true,
    }),
  ],
});
