/**
 * Browser + CLI Side-by-Side Recorder
 *
 * Creates a GIF showing:
 * - Left panel: Schema code (simulated editor)
 * - Right panel: Live OpenAPI/GraphQL output (browser-style)
 *
 * Usage: node browser-recorder.js [output-dir]
 * Requirements: npm install playwright gifencoder pngjs
 */

const { chromium } = require('playwright');
const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const OUTPUT_DIR = process.argv[2] || './docs/gifs';
const FRAMES_DIR = path.join(OUTPUT_DIR, 'frames');
const PROJECT_DIR = '/tmp/alab-browser-demo';

// Schema evolution steps
const STEPS = [
  {
    title: 'Step 1: Create basic schema',
    schema: `// schemas/blog/post.js

export default table({
  id: col.id(),
  title: col.text(200),
});`,
    format: 'openapi'
  },
  {
    title: 'Step 2: Add content fields',
    schema: `// schemas/blog/post.js

export default table({
  id: col.id(),
  title: col.text(200),
  content: col.text(),
  slug: col.text(100).unique(),
});`,
    format: 'openapi'
  },
  {
    title: 'Step 3: Add relationships',
    schema: `// schemas/blog/post.js

export default table({
  id: col.id(),
  title: col.text(200),
  content: col.text(),
  slug: col.text(100).unique(),
  author_id: col.fk("auth.user"),
  is_published: col.flag(false),
}).timestamps();`,
    format: 'openapi'
  },
  {
    title: 'Step 4: Export as GraphQL',
    schema: `// schemas/blog/post.js

export default table({
  id: col.id(),
  title: col.text(200),
  content: col.text(),
  slug: col.text(100).unique(),
  author_id: col.fk("auth.user"),
  is_published: col.flag(false),
}).timestamps();`,
    format: 'graphql'
  }
];

// HTML template for the split view
const HTML_TEMPLATE = `
<!DOCTYPE html>
<html>
<head>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: 'SF Mono', 'Consolas', 'Monaco', monospace;
      background: #0d1117;
      color: #c9d1d9;
      height: 100vh;
      overflow: hidden;
    }
    .container {
      display: flex;
      height: 100vh;
    }
    .panel {
      flex: 1;
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
    .panel-left {
      background: #161b22;
      border-right: 1px solid #30363d;
    }
    .panel-right {
      background: #0d1117;
    }
    .header {
      background: #21262d;
      padding: 12px 16px;
      border-bottom: 1px solid #30363d;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .header-icon {
      font-size: 16px;
    }
    .header-title {
      font-size: 13px;
      font-weight: 600;
      color: #c9d1d9;
    }
    .header-subtitle {
      font-size: 11px;
      color: #8b949e;
      margin-left: auto;
    }
    .content {
      flex: 1;
      overflow: auto;
      padding: 16px;
    }
    pre {
      font-size: 13px;
      line-height: 1.6;
      white-space: pre-wrap;
      word-break: break-word;
    }
    .editor-line-numbers {
      color: #6e7681;
      user-select: none;
      text-align: right;
      padding-right: 16px;
      border-right: 1px solid #30363d;
      margin-right: 16px;
    }
    .editor-content {
      display: flex;
    }
    /* Syntax highlighting */
    .keyword { color: #ff7b72; }
    .string { color: #a5d6ff; }
    .function { color: #d2a8ff; }
    .comment { color: #8b949e; }
    .property { color: #79c0ff; }
    .number { color: #79c0ff; }
    /* JSON/YAML highlighting */
    .json-key { color: #7ee787; }
    .json-string { color: #a5d6ff; }
    .json-number { color: #79c0ff; }
    .json-bool { color: #ff7b72; }
    /* Status bar */
    .status-bar {
      background: #238636;
      color: white;
      padding: 8px 16px;
      font-size: 12px;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .status-bar.graphql {
      background: #e535ab;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="panel panel-left">
      <div class="header">
        <span class="header-icon">📝</span>
        <span class="header-title">Schema Editor</span>
        <span class="header-subtitle" id="step-title">Step 1</span>
      </div>
      <div class="content">
        <pre id="schema"></pre>
      </div>
    </div>
    <div class="panel panel-right">
      <div class="header">
        <span class="header-icon" id="output-icon">🔄</span>
        <span class="header-title" id="output-title">OpenAPI Output</span>
        <span class="header-subtitle">Live Preview</span>
      </div>
      <div class="content">
        <pre id="output"></pre>
      </div>
      <div class="status-bar" id="status-bar">
        <span>✓</span>
        <span id="status-text">Schema exported successfully</span>
      </div>
    </div>
  </div>
</body>
</html>
`;

function highlightJS(code) {
  return code
    .replace(/(\/\/.*)/g, '<span class="comment">$1</span>')
    .replace(/\b(export|default|const|let|var|function|return|if|else)\b/g, '<span class="keyword">$1</span>')
    .replace(/\b(table|col)\b/g, '<span class="function">$1</span>')
    .replace(/(\.[a-zA-Z_]+)\(/g, '<span class="property">$1</span>(')
    .replace(/"([^"]*)"/g, '<span class="string">"$1"</span>')
    .replace(/\b(\d+)\b/g, '<span class="number">$1</span>')
    .replace(/\b(true|false)\b/g, '<span class="keyword">$1</span>');
}

function highlightJSON(json) {
  try {
    const obj = JSON.parse(json);
    const pretty = JSON.stringify(obj, null, 2);
    return pretty
      .replace(/"([^"]+)":/g, '<span class="json-key">"$1"</span>:')
      .replace(/: "([^"]*)"/g, ': <span class="json-string">"$1"</span>')
      .replace(/: (\d+)/g, ': <span class="json-number">$1</span>')
      .replace(/: (true|false)/g, ': <span class="json-bool">$1</span>');
  } catch {
    return json;
  }
}

function highlightGraphQL(gql) {
  return gql
    .replace(/\b(type|input|enum|interface|scalar|query|mutation|subscription)\b/g, '<span class="keyword">$1</span>')
    .replace(/\b(ID|String|Int|Float|Boolean)\b/g, '<span class="function">$1</span>')
    .replace(/(#.*)/g, '<span class="comment">$1</span>');
}

async function setup() {
  console.log('Setting up demo project...');
  execSync(`rm -rf ${PROJECT_DIR}`, { stdio: 'pipe' });
  execSync(`mkdir -p ${PROJECT_DIR}`, { stdio: 'pipe' });
  execSync(`cd ${PROJECT_DIR} && alab init`, { stdio: 'pipe' });
  execSync(`mkdir -p ${PROJECT_DIR}/schemas/blog`, { stdio: 'pipe' });
  fs.mkdirSync(FRAMES_DIR, { recursive: true });
}

async function exportSchema(format) {
  const outputFile = `/tmp/export-output.${format === 'graphql' ? 'graphql' : 'json'}`;
  try {
    execSync(`cd ${PROJECT_DIR} && alab export -f ${format} -o ${outputFile}`, { stdio: 'pipe' });
    return fs.readFileSync(outputFile, 'utf-8');
  } catch (e) {
    return `Error: ${e.message}`;
  }
}

async function main() {
  await setup();

  console.log('Launching browser...');
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.setViewportSize({ width: 1920, height: 1080 });
  await page.setContent(HTML_TEMPLATE);

  let frameIndex = 0;

  for (const step of STEPS) {
    console.log(`Recording: ${step.title}`);

    // Write schema file
    const schemaPath = path.join(PROJECT_DIR, 'schemas/blog/post.js');
    fs.writeFileSync(schemaPath, step.schema.replace('// schemas/blog/post.js\n\n', ''));

    // Export
    const output = await exportSchema(step.format);

    // Update page
    await page.evaluate(({ schema, output, title, format }) => {
      document.getElementById('step-title').textContent = title;
      document.getElementById('output-title').textContent = format === 'graphql' ? 'GraphQL Schema' : 'OpenAPI Specification';
      document.getElementById('output-icon').textContent = format === 'graphql' ? '⬡' : '📄';
      document.getElementById('status-bar').className = format === 'graphql' ? 'status-bar graphql' : 'status-bar';
      document.getElementById('status-text').textContent = format === 'graphql'
        ? 'GraphQL schema generated'
        : 'OpenAPI spec generated';
    }, { schema: step.schema, output, title: step.title, format: step.format });

    // Highlight and set content
    const highlightedSchema = highlightJS(step.schema);
    const highlightedOutput = step.format === 'graphql' ? highlightGraphQL(output) : highlightJSON(output);

    await page.evaluate(({ schema, output }) => {
      document.getElementById('schema').innerHTML = schema;
      document.getElementById('output').innerHTML = output;
    }, { schema: highlightedSchema, output: highlightedOutput });

    // Capture multiple frames for this step (creates pause effect)
    for (let i = 0; i < 30; i++) { // 30 frames at ~10fps = 3 seconds per step
      const framePath = path.join(FRAMES_DIR, `frame-${String(frameIndex++).padStart(4, '0')}.png`);
      await page.screenshot({ path: framePath });
    }
  }

  await browser.close();

  console.log(`\nFrames saved to ${FRAMES_DIR}/`);
  console.log('\nTo create GIF, run:');
  console.log(`  ffmpeg -framerate 10 -i "${FRAMES_DIR}/frame-%04d.png" -vf "fps=10,scale=960:-1:flags=lanczos" -loop 0 "${OUTPUT_DIR}/live-browser.gif"`);
  console.log('\nOr for better quality:');
  console.log(`  gifski --fps 10 --width 960 "${FRAMES_DIR}"/frame-*.png -o "${OUTPUT_DIR}/live-browser.gif"`);
}

main().catch(console.error);
