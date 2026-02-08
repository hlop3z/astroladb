#!/usr/bin/env node
/**
 * Test all generated API servers
 * Usage: node test-all.mjs [framework] [--auto]
 *
 * Note: In auto mode, dependencies must be installed first:
 *   - Python: uv add fastapi
 *   - TypeScript: npm install
 *   - Rust: cargo build
 *   - Go: go mod tidy
 */

import { spawn } from "child_process";
import { setTimeout as sleep } from "timers/promises";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const __dirname = dirname(fileURLToPath(import.meta.url));
const SCRIPT_DIR = __dirname;
const OUTPUT_DIR = join(SCRIPT_DIR, "generated");

// Parse args
const args = process.argv.slice(2);
const framework = args.find((a) => !a.startsWith("--")) || "all";
const autoMode = args.includes("--auto");

// Colors
const colors = {
  reset: "\x1b[0m",
  red: "\x1b[31m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  blue: "\x1b[34m",
};

const log = {
  info: (msg) => console.log(`${colors.green}${msg}${colors.reset}`),
  error: (msg) => console.log(`${colors.red}${msg}${colors.reset}`),
  warn: (msg) => console.log(`${colors.yellow}${msg}${colors.reset}`),
  blue: (msg) => console.log(`${colors.blue}${msg}${colors.reset}`),
};

// Server management
const servers = [];

const cleanup = () => {
  if (servers.length > 0) {
    log.warn("\n🛑 Stopping servers...");
    servers.forEach((proc) => {
      try {
        process.kill(proc.pid);
        log.warn(`  Stopped PID ${proc.pid}`);
      } catch {}
    });
  }
  process.exit(0);
};

process.on("SIGINT", cleanup);
process.on("SIGTERM", cleanup);

// Wait for server to be ready
async function waitForServer(url, maxAttempts = 30) {
  log.warn(`  Waiting for ${url}...`);
  for (let i = 0; i < maxAttempts; i++) {
    try {
      await fetch(url);
      log.info("  ✓ Server ready!");
      return true;
    } catch {
      await sleep(1000);
    }
  }
  log.error(`  ✗ Server failed to start`);
  return false;
};

// Start server in background
async function startServer(name, port, cmd, cwd) {
  log.blue(`🚀 Starting ${name} server on port ${port}...`);

  const proc = spawn(cmd, {
    cwd,
    stdio: "ignore",
    shell: true,
    detached: false,
  });

  proc.on("error", (err) => {
    log.error(`Failed to start ${name}: ${err.message}`);
  });

  servers.push(proc);
  log.warn(`  Started with PID ${proc.pid}`);
  return waitForServer(`http://localhost:${port}/`);
}

// Test endpoint
async function testEndpoint(method, url, data, expectedStatus, description) {
  log.blue(`  → ${description}`);
  try {
    const res = await fetch(url, {
      method,
      headers: data ? { "Content-Type": "application/json" } : {},
      body: data ? JSON.stringify(data) : undefined,
    });
    const status = res.status;

    if ([expectedStatus, 501, 404].includes(status)) {
      const symbol = status === expectedStatus ? "✓" : "⚠";
      const color = status === expectedStatus ? colors.green : colors.yellow;
      console.log(`    ${color}${symbol} ${method} ${url} → ${status}${colors.reset}`);
      return true;
    } else {
      log.error(`    ✗ ${method} ${url} → ${status} (expected ${expectedStatus})`);
      return false;
    }
  } catch (err) {
    log.error(`    ✗ ${method} ${url} → ${err.message}`);
    return false;
  }
}

// Test CRUD
async function testCRUD(name, baseUrl, resource) {
  log.info(`\n${"═".repeat(39)}`);
  log.info(`  ${name}`);
  log.info("═".repeat(39));

  await testEndpoint("GET", `${baseUrl}/`, null, 200, "Health check");
  await testEndpoint("GET", `${baseUrl}/${resource}`, null, 200, `List ${resource}`);
  await testEndpoint("POST", `${baseUrl}/${resource}`, { name: "Test" }, 201, `Create ${resource}`);

  const testId = "123e4567-e89b-12d3-a456-426614174000";
  await testEndpoint("GET", `${baseUrl}/${resource}/${testId}`, null, 200, `Get by ID`);
  await testEndpoint("PATCH", `${baseUrl}/${resource}/${testId}`, { name: "Updated" }, 200, `Update`);
  await testEndpoint("DELETE", `${baseUrl}/${resource}/${testId}`, null, 204, `Delete`);
}

// Framework testers
const frameworks = {
  async python() {
    const port = 8000;
    const url = `http://localhost:${port}`;

    if (autoMode) {
      const started = await startServer(
        "python",
        port,
        "uv run fastapi dev main.py",
        join(OUTPUT_DIR, "python-fastapi")
      );
      if (!started) return false;
    } else if (!(await checkServer(url, "Python"))) {
      return false;
    }

    await testCRUD("Python (FastAPI)", `${url}/auth`, "users");
  },

  async typescript() {
    const port = 3000;
    const url = `http://localhost:${port}`;

    if (autoMode) {
      const started = await startServer(
        "typescript",
        port,
        "npx tsx server.ts",
        join(OUTPUT_DIR, "typescript-trpc")
      );
      if (!started) return false;
    } else if (!(await checkServer(url))) {
      return false;
    }

    log.blue("Testing tRPC endpoints...");
    await testEndpoint("GET", `${url}/users.list`, null, 200, "tRPC list query");
  },

  async rust() {
    const port = 3000;
    const url = `http://localhost:${port}`;

    if (autoMode) {
      const started = await startServer(
        "rust",
        port,
        "cargo run --release",
        join(OUTPUT_DIR, "rust-axum")
      );
      if (!started) return false;
    } else if (!(await checkServer(url))) {
      return false;
    }

    await testCRUD("Rust (Axum)", `${url}/auth`, "users");
  },

  async go() {
    const port = 3000;
    const url = `http://localhost:${port}`;

    if (autoMode) {
      const started = await startServer(
        "go",
        port,
        "go run main.go",
        join(OUTPUT_DIR, "go-chi")
      );
      if (!started) return false;
    } else if (!(await checkServer(url))) {
      return false;
    }

    await testCRUD("Go (Chi)", `${url}/auth`, "users");
  },
};

async function checkServer(url, framework) {
  try {
    await fetch(url);
    return true;
  } catch {
    log.error(`❌ Server not running at ${url}`);
    if (autoMode) {
      log.warn(`💡 Make sure dependencies are installed for ${framework}`);
      log.warn(`   See generated README for installation instructions`);
    } else {
      log.warn(`💡 Start manually or use --auto flag (after installing deps)`);
    }
    return false;
  }
}

// Main
(async () => {
  log.info("🧪 Testing Generated APIs\n");

  try {
    if (framework === "all") {
      for (const [name, test] of Object.entries(frameworks)) {
        try {
          await test();
        } catch (err) {
          log.error(`Error testing ${name}: ${err.message}`);
        }
      }
    } else if (frameworks[framework]) {
      await frameworks[framework]();
    } else {
      log.error(`Unknown framework: ${framework}`);
      console.log("\nUsage: node test-all.mjs [python|typescript|rust|go|all] [--auto]");
      process.exit(1);
    }

    log.info("\n✨ Testing complete!");
    cleanup();
  } catch (err) {
    log.error(`\nError: ${err.message}`);
    cleanup();
  }
})();
