import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
const root = new URL("../", import.meta.url);
export function generateTheme(tokens) {
  const declarations = (data, prefix = "") =>
    Object.entries(data)
      .map(([key, value]) => `  --flow-${prefix}${key}: ${value};`)
      .join("\n");
  let css = "/* Generated from theme/tokens.json by scripts/theme.mjs. */\n";
  css += `:root {\n${declarations(tokens.shape, "shape-")}\n${declarations(tokens.motion, "motion-")}\n}\n`;
  css += `:root {\n${declarations(tokens.typography, "type-")}\n${declarations(tokens.spacing, "space-")}\n${declarations(tokens.elevation, "elevation-")}\n${declarations(tokens.focus, "focus-")}\n}\n`;
  for (const [scheme, values] of Object.entries(tokens.schemes)) {
    css += `:root[data-theme="${scheme}"] {\n  color-scheme: ${scheme};\n${declarations(values)}\n}\n`;
    for (const [palette, schemes] of Object.entries(tokens.palettes))
      css += `:root[data-theme="${scheme}"][data-palette="${palette}"] {\n${declarations(schemes[scheme])}\n  --flow-accent: var(--flow-primary);\n}\n`;
  }
  return css;
}
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const css = generateTheme(
    JSON.parse(readFileSync(new URL("src/theme/tokens.json", root), "utf8")),
  );
  const path = new URL("src/theme/generated.css", root);
  if (process.argv.includes("--check")) {
    if (readFileSync(path, "utf8") !== css)
      throw new Error("Regenerate theme CSS with node scripts/theme.mjs");
  } else writeFileSync(path, css);
}
