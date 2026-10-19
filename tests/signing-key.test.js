import { afterEach, beforeAll, beforeEach, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { existsSync, mkdtempSync, readFileSync, rmSync, statSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { createSigningKey, gpgTempRoot } from "./helpers.js";

const script = new URL("../scripts/signing-key.sh", import.meta.url).pathname;
let key;
let otherKey;
let directory;
let keyrings;

beforeAll(() => {
  key = createSigningKey("Release signing fixture");
  otherKey = createSigningKey("Unrelated signing fixture");
}, 30_000);
beforeEach(() => {
  directory = mkdtempSync(join(gpgTempRoot, "release-import-"));
  keyrings = new Set();
});
afterEach(() => {
  try {
    for (const home of keyrings) {
      if (!existsSync(home)) continue;
      const result = run("cleanup", { GNUPGHOME: home });
      if (result.exitCode !== 0) throw new Error(result.stderr.toString());
    }
  } finally {
    rmSync(directory, { recursive: true });
  }
});

function run(command, env = {}) {
  return Bun.spawnSync(["bash", script, command], {
    env: { ...process.env, RUNNER_TEMP: directory, GITHUB_OUTPUT: join(directory, "outputs"), ...env },
    stdout: "pipe", stderr: "pipe",
  });
}

function importKey(privateKey, env = {}) {
  writeFileSync(join(directory, "outputs"), "");
  const result = run("import", { PGP_PRIVATE_KEY: privateKey, ...env });
  const outputs = Object.fromEntries(readFileSync(join(directory, "outputs"), "utf8").trim().split("\n")
    .filter(Boolean).map((line) => [line.slice(0, line.indexOf("=")), line.slice(line.indexOf("=") + 1)]));
  if (outputs.gnupghome) keyrings.add(outputs.gnupghome);
  return { result, ...outputs };
}

function gpg(home, args) {
  return Bun.spawnSync(["gpg", "--batch", ...args], {
    env: { ...process.env, GNUPGHOME: home },
    stdout: "pipe", stderr: "pipe",
  });
}

test("imports one private primary key with multiple subkeys into a private isolated keyring", () => {
  const imported = importKey(key.privateKey);
  expect(imported.result.exitCode).toBe(0);
  expect(imported.fingerprint).toBe(key.fingerprint);
  expect(statSync(imported.gnupghome).mode & 0o777).toBe(0o700);
  expect(imported.result.stdout.toString() + imported.result.stderr.toString())
    .not.toContain("-----BEGIN PGP PRIVATE KEY BLOCK-----");
});

test("the imported key signs a checksum file and tampering invalidates its signature", () => {
  const imported = importKey(key.privateKey);
  expect(imported.result.exitCode).toBe(0);
  const checksums = join(directory, "checksums.txt");
  const signature = `${checksums}.sig`;
  writeFileSync(checksums, `${createHash("sha256").update("fixture artifact").digest("hex")}  fixture.tar.gz\n`);
  expect(gpg(imported.gnupghome, [
    "--local-user", imported.fingerprint, "--output", signature, "--detach-sign", checksums,
  ]).exitCode).toBe(0);
  expect(gpg(imported.gnupghome, ["--verify", signature, checksums]).exitCode).toBe(0);
  writeFileSync(checksums, "tampered checksum\n");
  expect(gpg(imported.gnupghome, ["--verify", signature, checksums]).exitCode).not.toBe(0);
});

test("an existing caller keyring is not modified", () => {
  const caller = importKey(otherKey.privateKey);
  expect(caller.result.exitCode).toBe(0);
  const before = gpg(caller.gnupghome, ["--with-colons", "--list-secret-keys"]).stdout.toString();
  const imported = importKey(key.privateKey, { GNUPGHOME: caller.gnupghome });
  expect(imported.result.exitCode).toBe(0);
  expect(imported.gnupghome).not.toBe(caller.gnupghome);
  expect(gpg(caller.gnupghome, ["--with-colons", "--list-secret-keys"]).stdout.toString()).toBe(before);
});

test("rejects empty, invalid, and public-only key inputs", () => {
  for (const input of ["", "not an OpenPGP key", key.publicKey]) {
    const imported = importKey(input);
    expect(imported.result.exitCode).not.toBe(0);
    expect(imported.fingerprint).toBeUndefined();
  }
});

test("rejects multiple primary private keys rather than selecting one silently", () => {
  const imported = importKey(`${key.privateKey}\n${otherKey.privateKey}`);
  expect(imported.result.exitCode).not.toBe(0);
  expect(imported.result.stderr.toString()).toContain("exactly one private primary key");
  expect(imported.fingerprint).toBeUndefined();
});

test("cleanup removes only the imported keyring", () => {
  const imported = importKey(key.privateKey);
  expect(imported.result.exitCode).toBe(0);
  const unrelated = join(directory, "unrelated");
  writeFileSync(unrelated, "preserve");
  expect(run("cleanup", { GNUPGHOME: directory }).exitCode).not.toBe(0);
  expect(readFileSync(unrelated, "utf8")).toBe("preserve");
  expect(run("cleanup", { GNUPGHOME: imported.gnupghome }).exitCode).toBe(0);
  expect(existsSync(imported.gnupghome)).toBe(false);
  expect(readFileSync(unrelated, "utf8")).toBe("preserve");
});
