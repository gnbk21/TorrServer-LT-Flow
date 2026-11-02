import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import tokens from "./tokens.json";
import { generateTheme } from "../../scripts/theme.mjs";
function luminance(hex: string) {
  const c = [1, 3, 5]
    .map((at) => parseInt(hex.slice(at, at + 2), 16) / 255)
    .map((v) => (v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4));
  return c[0]! * 0.2126 + c[1]! * 0.7152 + c[2]! * 0.0722;
}
function contrast(a: string, b: string) {
  const x = luminance(a),
    y = luminance(b);
  return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05);
}
describe("expressive theme contract", () => {
  it("keeps generated CSS synchronized", () =>
    expect(readFileSync("src/theme/generated.css", "utf8")).toBe(
      generateTheme(tokens),
    ));
  for (const scheme of ["light", "dark"] as const)
    for (const [name, colors] of Object.entries(tokens.palettes)) {
      it(`${name}/${scheme} has readable surfaces and controls`, () => {
        const t = { ...tokens.schemes[scheme], ...colors[scheme] };
        for (const surface of [t.bg, t.surface, t["surface-hover"], t.input]) {
          expect(contrast(t.text, surface)).toBeGreaterThanOrEqual(4.5);
          expect(contrast(t.muted, surface)).toBeGreaterThanOrEqual(4.5);
          expect(contrast(t.primary, surface)).toBeGreaterThanOrEqual(4.5);
          for (const role of [t.error, t.warning, t.success])
            expect(contrast(role, surface)).toBeGreaterThanOrEqual(4.5);
          expect(contrast(t.border, surface)).toBeGreaterThanOrEqual(3);
        }
        expect(contrast(t.primary, t["on-primary"])).toBeGreaterThanOrEqual(
          4.5,
        );
        expect(
          contrast(t["primary-container"], t["on-container"]),
        ).toBeGreaterThanOrEqual(4.5);
      });
    }
});
