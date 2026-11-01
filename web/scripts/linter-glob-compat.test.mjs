import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);
const estree = createRequire(require.resolve("@typescript-eslint/typescript-estree/package.json"));
const eslint = createRequire(require.resolve("eslint/package.json"));
const patched = estree("minimatch");
const legacy = eslint("minimatch");

test("the parser uses the patched matcher without changing ESLint's separate API", () => {
  assert.equal(estree("minimatch/package.json").version, "11.0.1");
  assert.match(eslint("minimatch/package.json").version, /^3\./);
  const matchLegacy = legacy.minimatch ?? legacy;
  // typescript-estree's allowDefaultProject consumer uses this named export
  // with dot:true. Exercise its file/glob contract across the major override.
  for (const [file, glob, expected] of [
    ["config.ts", "*.ts", true],
    [".config.ts", "*.ts", true],
    ["src/config.ts", "*.ts", false],
    ["src/Settings.tsx", "src/**/*.{ts,tsx}", true],
    ["src/Settings.css", "src/**/*.{ts,tsx}", false],
    ["node_modules/dep/index.js", "**/node_modules/**", true],
  ]) {
    assert.equal(patched.minimatch(file, glob, { dot: true }), expected);
    assert.equal(matchLegacy(file, glob, { dot: true }), expected);
  }
});
