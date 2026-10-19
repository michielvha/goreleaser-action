import { appendFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { createSigningKey } from "./helpers.js";

const key = createSigningKey("GoReleaser snapshot fixture");
const escaped = key.privateKey.replaceAll("%", "%25").replaceAll("\r", "%0D").replaceAll("\n", "%0A");
console.log(`::add-mask::${escaped}`);
appendFileSync(process.env.GITHUB_OUTPUT, `private-key<<PRIVATE_KEY\n${key.privateKey}\nPRIVATE_KEY\n`);
writeFileSync(join(process.env.RUNNER_TEMP, "release-fixture-public.asc"), key.publicKey);
