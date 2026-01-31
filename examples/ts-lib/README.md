# TypeScript Generator Example

Shows how to author alab generators in TypeScript with reusable modules, bundled to a single ES5-compatible JS file using [esbuild](https://esbuild.github.io/).

## Structure

```
src/
├── main.ts        # Entry point: uses gen() + render()
├── helpers.ts     # Type mapping, naming utilities
└── templates.ts   # Python code template builders
```

esbuild bundles everything into `generators/fastapi.js` — a single IIFE that alab's JS runtime (Goja) can execute.

## Usage

```bash
# Install esbuild
npm install

# Bundle TypeScript → generators/fastapi.js
npm run build

# Run the generator
alab gen run fastapi -o generated
```

## Why esbuild?

- Zero config for library bundling
- Single command, no plugins needed
- `--format=iife --target=es2015` produces Goja-compatible output
- Vite is overkill for this use case (it's a dev server + bundler for apps)
