import { expect, test } from "bun:test";

const action = Bun.YAML.parse(await Bun.file(new URL("../action.yml", import.meta.url)).text());

test("the action composes only Go setup, GPG import, and GoReleaser", () => {
  expect(action.runs.using).toBe("composite");
  expect(action.runs.steps.map((step) => step.uses.split("@")[0])).toEqual([
    "actions/setup-go",
    "crazy-max/ghaction-import-gpg",
    "goreleaser/goreleaser-action",
  ]);
  expect(action.runs.steps.every((step) => !step.run && !step["continue-on-error"])).toBe(true);
});

test("CI and unused versioning inputs are removed only from the next-major contract", () => {
  expect(Object.keys(action.inputs)).toEqual([
    "go-version", "github-token", "pgp-private-key", "goreleaser-version", "goreleaser-args",
  ]);
});

test("the imported key fingerprint and release credentials go to GoReleaser", () => {
  const signing = action.runs.steps[1];
  const release = action.runs.steps[2];
  expect(signing.with.gpg_private_key).toBe("${{ inputs.pgp-private-key }}");
  expect(release.env.GITHUB_TOKEN).toBe("${{ inputs.github-token }}");
  expect(release.env.GPG_FINGERPRINT).toBe("${{ steps.gpg.outputs.fingerprint }}");
  expect(release.with.args).toBe("${{ inputs.goreleaser-args }}");
  expect(action.inputs["goreleaser-args"].default).toBe("release --clean");
});

test("dependencies use immutable commit SHAs", () => {
  for (const step of action.runs.steps) {
    expect(step.uses).toMatch(/@[0-9a-f]{40}$/);
  }
});
