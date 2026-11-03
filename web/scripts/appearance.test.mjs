import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
const bootstrap = readFileSync(
  new URL("../public/appearance.js", import.meta.url),
  "utf8",
);
const run = (storage, dark = false) => {
  const root = { dataset: {}, style: {} };
  runInNewContext(bootstrap, {
    document: { documentElement: root },
    localStorage: storage,
    matchMedia: () => ({ matches: dark }),
  });
  return root;
};
test("first paint tolerates blocked/malformed preferences and follows system", () => {
  assert.equal(
    run(
      {
        getItem() {
          throw Error("blocked");
        },
      },
      true,
    ).dataset.theme,
    "dark",
  );
  const result = run({ getItem: () => "{broken" });
  assert.equal(result.dataset.theme, "light");
  assert.equal(result.style.fontSize, "100%");
  assert.equal(result.dataset.palette, "iris");
});
test("saved preference validation is identical before React loads", () => {
  const values = {
    "flow.ui.theme": '"dark"',
    "flow.ui.palette": '"lagoon"',
    "flow.ui.textScale": "200",
    "flow.ui.motion": '"reduce"',
    "flow.ui.density": '"compact"',
  };
  const result = run({ getItem: (key) => values[key] ?? null });
  assert.deepEqual(result.dataset, {
    theme: "dark",
    palette: "lagoon",
    density: "compact",
    motion: "reduce",
  });
  assert.equal(result.style.fontSize, "200%");
});
