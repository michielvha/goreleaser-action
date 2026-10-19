import { execFileSync, spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, rmSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";

// macOS's default temp path can exceed GnuPG's Unix socket length once nested.
export const gpgTempRoot = process.platform === "darwin" ? "/tmp" : tmpdir();

export function stopAgent(home) {
  const env = { ...process.env, GNUPGHOME: home };
  const socket = execFileSync("gpgconf", ["--list-dirs", "agent-socket"], { env, encoding: "utf8" }).trim();
  if (!existsSync(socket)) return;
  const info = execFileSync("gpg-connect-agent", ["--no-autostart", "GETINFO pid", "/bye"], {
    env, encoding: "utf8",
  });
  const pid = Number(/^D (\d+)$/m.exec(info)?.[1]);
  if (!Number.isInteger(pid) || pid <= 1) throw new Error("The fixture agent has no valid PID");
  try {
    process.kill(pid);
  } catch (error) {
    if (error.code !== "ESRCH") throw error;
  }
}

export function createSigningKey(name) {
  const directory = mkdtempSync(join(gpgTempRoot, "release-key-"));
  const env = { ...process.env, GNUPGHOME: directory };
  function gpg(args) {
    const result = spawnSync("gpg", ["--batch", "--pinentry-mode", "loopback", "--passphrase", "", ...args], {
      env, encoding: "utf8",
    });
    if (result.status !== 0) throw new Error(`Fixture key generation failed: ${result.stderr}`);
    return result.stdout;
  }
  try {
    gpg(["--quick-generate-key", `${name} <fixture@example.invalid>`, "ed25519", "cert", "0"]);
    const fingerprint = gpg(["--with-colons", "--list-secret-keys"])
      .split("\n").find((line) => line.startsWith("fpr:")).split(":")[9];
    gpg(["--quick-add-key", fingerprint, "ed25519", "sign", "0"]);
    gpg(["--quick-add-key", fingerprint, "cv25519", "encrypt", "0"]);
    return {
      fingerprint,
      privateKey: gpg(["--armor", "--export-secret-keys", fingerprint]),
      publicKey: gpg(["--armor", "--export", fingerprint]),
    };
  } finally {
    try {
      stopAgent(directory);
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  }
}
