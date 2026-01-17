/**
 * Automated Browser + Schema Recording
 *
 * Records the live export experience:
 * 1. Creates a schema file
 * 2. Exports to OpenAPI/GraphQL
 * 3. Shows the output in browser
 * 4. Captures screenshots at each step
 *
 * Usage: node live-export.js
 * Requires: playwright, alab CLI
 */

const { chromium } = require('playwright');
const { execSync, spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

const OUTPUT_DIR = process.env.OUTPUT_DIR || './recording-frames';
const PROJECT_DIR = process.env.PROJECT_DIR || '/tmp/alab-demo';

// Schema content to write progressively
const SCHEMA_STEPS = [
  {
    name: 'Initial schema',
    content: `// schemas/blog/post.js
export default table({
  id: col.id(),
});
`
  },
  {
    name: 'Add title field',
    content: `// schemas/blog/post.js
export default table({
  id: col.id(),
  title: col.text(200),
});
`
  },
  {
    name: 'Add content and author',
    content: `// schemas/blog/post.js
export default table({
  id: col.id(),
  title: col.text(200),
  content: col.text(),
  author_id: col.fk("auth.user"),
}).timestamps();
`
  },
  {
    name: 'Add published flag',
    content: `// schemas/blog/post.js
export default table({
  id: col.id(),
  title: col.text(200),
  content: col.text(),
  author_id: col.fk("auth.user"),
  is_published: col.flag(false),
}).timestamps();
`
  }
];

async function setup() {
  // Clean and init project
  execSync(`rm -rf ${PROJECT_DIR} && mkdir -p ${PROJECT_DIR}`, { stdio: 'inherit' });
  execSync(`cd ${PROJECT_DIR} && alab init`, { stdio: 'inherit' });
  execSync(`cd ${PROJECT_DIR} && alab table blog post`, { stdio: 'inherit' });

  // Create output directory
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
}

async function exportAndCapture(browser, page, step, index) {
  const schemaPath = path.join(PROJECT_DIR, 'schemas/blog/post.js');

  // Write schema
  fs.writeFileSync(schemaPath, step.content);
  console.log(`Step ${index + 1}: ${step.name}`);

  // Export to OpenAPI
  execSync(`cd ${PROJECT_DIR} && alab export -f openapi -o /tmp/openapi.json`, { stdio: 'pipe' });

  // Read and display in browser
  const openapi = fs.readFileSync('/tmp/openapi.json', 'utf-8');

  // Update page with new content
  await page.evaluate((content) => {
    document.getElementById('schema').textContent = content;
  }, step.content);

  await page.evaluate((content) => {
    document.getElementById('output').textContent = content;
  }, openapi);

  // Take screenshot
  await page.screenshot({
    path: path.join(OUTPUT_DIR, `frame-${String(index).padStart(3, '0')}.png`),
    fullPage: true
  });

  // Wait for visual effect
  await new Promise(r => setTimeout(r, 1000));
}

async function main() {
  console.log('Setting up demo project...');
  await setup();

  console.log('Launching browser...');
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.setViewportSize({ width: 1920, height: 1080 });

  // Create a split-view HTML page
  await page.setContent(`
    <!DOCTYPE html>
    <html>
    <head>
      <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
          font-family: 'Consolas', 'Monaco', monospace;
          background: #1e1e1e;
          color: #d4d4d4;
          display: flex;
          height: 100vh;
        }
        .panel {
          flex: 1;
          padding: 20px;
          overflow: auto;
        }
        .panel-left {
          background: #252526;
          border-right: 2px solid #3c3c3c;
        }
        .panel-right {
          background: #1e1e1e;
        }
        h2 {
          color: #569cd6;
          margin-bottom: 15px;
          font-size: 14px;
          text-transform: uppercase;
          letter-spacing: 1px;
        }
        pre {
          font-size: 13px;
          line-height: 1.5;
          white-space: pre-wrap;
        }
        .schema { color: #ce9178; }
        .output { color: #4ec9b0; }
      </style>
    </head>
    <body>
      <div class="panel panel-left">
        <h2>📝 Schema (schemas/blog/post.js)</h2>
        <pre id="schema" class="schema"></pre>
      </div>
      <div class="panel panel-right">
        <h2>📄 OpenAPI Output</h2>
        <pre id="output" class="output"></pre>
      </div>
    </body>
    </html>
  `);

  // Capture each step
  for (let i = 0; i < SCHEMA_STEPS.length; i++) {
    await exportAndCapture(browser, page, SCHEMA_STEPS[i], i);
  }

  await browser.close();

  console.log(`\nScreenshots saved to ${OUTPUT_DIR}/`);
  console.log('Convert to GIF with:');
  console.log(`  ffmpeg -framerate 1 -i ${OUTPUT_DIR}/frame-%03d.png -vf "fps=1" output.gif`);
}

main().catch(console.error);
