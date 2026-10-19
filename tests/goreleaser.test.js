import { afterEach, beforeEach, expect, test } from "bun:test";
import { chmodSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const script = new URL("../scripts/goreleaser.sh", import.meta.url).pathname;
const action = Bun.YAML.parse(await Bun.file(new URL("../action.yml", import.meta.url)).text());
let directory;
let bin;
let cache;

beforeEach(() => {
  directory = mkdtempSync(join(tmpdir(), "goreleaser-test-"));
  bin = join(directory, "bin");
  cache = join(directory, "cache");
  mkdirSync(bin);
  writeFileSync(join(bin, "goreleaser"), `#!/usr/bin/env node
require("node:fs").appendFileSync(process.env.ARGV_LOG, JSON.stringify(process.argv.slice(2)) + "\\n");
process.exit(Number(process.env.FIXTURE_EXIT_CODE || 0));
`);
  chmodSync(join(bin, "goreleaser"), 0o755);
});
afterEach(() => rmSync(directory, { recursive: true }));

function run(command, env = {}) {
  return Bun.spawnSync(["bash", script, command], {
    env: {
      ...process.env, PATH: `${bin}:${process.env.PATH}`, CACHE_DIRECTORY: cache, RUNNER_TEMP: directory,
      GITHUB_OUTPUT: join(directory, "outputs"), ARGV_LOG: join(directory, "arguments"), ...env,
    },
    stdout: "pipe", stderr: "pipe",
  });
}

function succeeds(result) {
  if (result.exitCode !== 0) {
    throw new Error(result.stderr.toString() || result.stdout.toString());
  }
}

function argumentsReceived() {
  const raw = readFileSync(join(directory, "arguments"), "utf8");
  const calls = raw.trim().split("\n");
  if (calls.length !== 1) throw new Error(`Expected one CLI invocation, received:\n${raw}`);
  return JSON.parse(calls[0]);
}

test("exact stable versions enable caching; ranges and channels remain fresh", () => {
  for (const version of [action.inputs["goreleaser-version"].default, "v2.18.3", "2.18.3"]) {
    writeFileSync(join(directory, "outputs"), "");
    expect(run("prepare", { REQUESTED_VERSION: version }).exitCode).toBe(0);
    expect(readFileSync(join(directory, "outputs"), "utf8")).toBe("cacheable=true\n");
  }
  for (const version of ["~> v2", "latest", "v2.19.0-rc.1"]) {
    writeFileSync(join(directory, "outputs"), "");
    const result = run("prepare", { REQUESTED_VERSION: version });
    expect(result.exitCode).toBe(0);
    expect(readFileSync(join(directory, "outputs"), "utf8")).toBe("cacheable=false\n");
    expect(result.stdout.toString()).toContain("installed fresh");
  }
});

test("a cold install is staged and a warm cache runs without the installer binary", () => {
  expect(run("stage", { CACHE_HIT: "false" }).exitCode).toBe(0);
  const executable = readFileSync(join(cache, "goreleaser"));
  rmSync(join(bin, "goreleaser"));
  writeFileSync(join(directory, "outputs"), "");
  expect(run("stage", { CACHE_HIT: "true" }).exitCode).toBe(0);
  expect(readFileSync(join(cache, "goreleaser"))).toEqual(executable);
  expect(readFileSync(join(directory, "outputs"), "utf8")).toBe("cache-hit=true\n");
  succeeds(run("run", { GORELEASER_ARGS: "release --clean" }));
  expect(argumentsReceived()).toEqual(["release", "--clean"]);
});

test("a damaged restored cache is an error, not a successful cache hit", () => {
  const result = run("stage", { CACHE_HIT: "true" });
  expect(result.exitCode).not.toBe(0);
  expect(result.stderr.toString()).toContain("cache has no executable");
});

test("quoted arguments are preserved and shell syntax is never executed", () => {
  expect(run("stage").exitCode).toBe(0);
  const sentinel = join(directory, "should-not-exist");
  const args = `release --config "directory with spaces/config.yml" --release-notes 'notes file.md' ';' '$(touch ${sentinel})'`;
  succeeds(run("run", { GORELEASER_ARGS: args }));
  expect(argumentsReceived()).toEqual([
    "release", "--config", "directory with spaces/config.yml", "--release-notes", "notes file.md",
    ";", `$(touch ${sentinel})`,
  ]);
  expect(existsSync(sentinel)).toBe(false);
});

test("a failing executable fails the release step", () => {
  expect(run("stage").exitCode).toBe(0);
  expect(run("run", { GORELEASER_ARGS: "release --clean", FIXTURE_EXIT_CODE: "42" }).exitCode).not.toBe(0);
});

test("empty and malformed arguments fail without invoking the CLI", () => {
  expect(run("stage").exitCode).toBe(0);
  for (const args of ["", "   ", '""', 'release --config "unfinished']) {
    expect(run("run", { GORELEASER_ARGS: args }).exitCode).not.toBe(0);
    expect(existsSync(join(directory, "arguments"))).toBe(false);
  }
});

test("many arguments still invoke the CLI exactly once", () => {
  expect(run("stage").exitCode).toBe(0);
  const args = ["release", ...Array.from({ length: 3000 }, (_, i) => `argument-${i}`)];
  succeeds(run("run", { GORELEASER_ARGS: args.join(" ") }));
  expect(argumentsReceived()).toEqual(args);
});

test("external action dependencies remain SHA-pinned", () => {
  for (const step of action.runs.steps.filter((step) => step.uses)) {
    expect(step.uses).toMatch(/@[0-9a-f]{40}$/);
  }
});
