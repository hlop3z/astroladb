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
      logo: {
        src: "./src/assets/logo.png",
        replacesTitle: false,
      },
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
