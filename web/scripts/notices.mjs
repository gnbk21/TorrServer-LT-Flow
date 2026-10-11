import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import { createHash } from "node:crypto";

// Distribute the original license text even when minification strips comments.
// Cover the production dependency tree, including lazy chunks such as HLS/QR.
const webRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const manifest = JSON.parse(
  fs.readFileSync(path.join(webRoot, "package.json"), "utf8"),
);
const visited = new Map();
const packages = new Map();
function collect(name, from) {
  const require = createRequire(path.join(from, "package.json"));
  let file;
  try {
    file = require.resolve(`${name}/package.json`);
  } catch {
    file = require.resolve(name);
  }
  let directory = path.dirname(file);
  let metadata;
  while (directory !== path.dirname(directory)) {
    const candidate = path.join(directory, "package.json");
    if (fs.existsSync(candidate)) {
      const parsed = JSON.parse(fs.readFileSync(candidate, "utf8"));
      if (parsed.name === name) {
        metadata = parsed;
        break;
      }
    }
    directory = path.dirname(directory);
  }
  if (!metadata) throw new Error(`Cannot locate license package: ${name}`);
  const key = `${metadata.name}@${metadata.version}`;
  if (visited.has(key)) return key;
  const files = fs
    .readdirSync(directory)
    .filter((file) => /^(licen[cs]e|copying|notice)(?:[.-]|$)/i.test(file));
  let text = files
    .filter((file) => fs.statSync(path.join(directory, file)).isFile())
    .sort()
    .map((file) => fs.readFileSync(path.join(directory, file), "utf8"))
    .join("\n\n");
  const supplement = path.join(
    webRoot,
    "licenses",
    `${key.replaceAll("/", "__")}.txt`,
  );
  if (!text && fs.existsSync(supplement))
    text = fs.readFileSync(supplement, "utf8");
  if (!text) throw new Error(`Missing distributed license text: ${key}`);
  visited.set(key, text);
  const record = { name: metadata.name, version: metadata.version, license: typeof metadata.license === "string" ? metadata.license : null,
    notice_sha256: createHash("sha256").update(text).digest("hex"), dependencies: [] };
  packages.set(key, record);
  for (const dependency of Object.keys(metadata.dependencies || {}))
    record.dependencies.push(collect(dependency, directory));
  record.dependencies.sort();
  return key;
}
const roots = Object.keys(manifest.dependencies).map(name => collect(name, webRoot)).sort();
const notices = [...visited]
  .sort(([a], [b]) => a.localeCompare(b))
  .map(([name, text]) => `===== ${name} =====\n\n${text.trim()}\n`)
  .join("\n");
fs.writeFileSync(
  path.join(webRoot, "build", "THIRD_PARTY_NOTICES.txt"),
  "TorrServer-Flow web dependency license notices\n\n" + notices,
);
fs.writeFileSync(path.join(webRoot, "build", "DEPENDENCIES.json"), JSON.stringify({ schema_version: 1, roots,
  packages: [...packages].sort(([a],[b]) => a.localeCompare(b)).map(([key, value]) => ({ key, ...value })) }, null, 2)+"\n");
console.log(
  `Preserved license notices for ${visited.size} production dependency packages.`,
);
