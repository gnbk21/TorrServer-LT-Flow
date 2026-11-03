import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { upgradeFixture } from "./upgrade-fixture";
import { readFileSync } from "node:fs";

test("unmocked APIs stay inside the isolated preview", async ({ request }) => {
  const response = await request.get("/flow/preparation");
  expect(response.status()).toBe(404);
  expect(await response.json()).toEqual({
    error: "endpoint omitted by isolated browser fixture",
  });
});

test("short zoom-equivalent viewports keep all main headings unobscured", async ({
  page,
}) => {
  await upgradeFixture(page, 2);
  await page.setViewportSize({ width: 320, height: 201 });
  for (const route of ["/", "/torrents", "/add", "/settings"]) {
    await page.goto(`/#${route}`);
    const heading = page.locator("#main h1");
    await expect(heading).toBeVisible();
    await heading.scrollIntoViewIfNeeded();
    await expect
      .poll(() =>
        heading.evaluate((element) => {
          const r = element.getBoundingClientRect();
          const at = document.elementFromPoint(
            r.x + r.width / 2,
            r.y + r.height / 2,
          );
          return at === element || element.contains(at);
        }),
      )
      .toBe(true);
  }
});

test("dashboard shortcut paints loading before a cold Library finishes", async ({
  page,
}) => {
  await upgradeFixture(page, 1);
  await page.addInitScript(() => localStorage.setItem("i18nextLng", "en"));
  let release!: () => void;
  const held = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route("**/assets/Torrents-*.js", async (route) => {
    await held;
    await route.continue();
  });
  try {
    await page.goto("/#/");
    await expect(
      page.getByRole("heading", { name: "Ready when you are" }),
    ).toBeVisible();
    await page
      .locator("#main")
      .getByRole("link", { name: "Torrents", exact: true })
      .click();
    await expect(page.locator("#main").getByRole("status")).toBeVisible();
    release();
    await expect(page.getByRole("article")).toHaveCount(1);
  } finally {
    release();
  }
});

for (const palette of ["iris", "lagoon", "rose"])
  for (const theme of ["light", "dark"]) {
    test(`expressive contrast and commands ${palette} ${theme}`, async ({
      page,
    }, info) => {
      await upgradeFixture(page, 2);
      await page.addInitScript(
        ({ palette, theme }) => {
          localStorage.setItem("flow.ui.palette", JSON.stringify(palette));
          localStorage.setItem("flow.ui.theme", JSON.stringify(theme));
          localStorage.setItem("i18nextLng", "en");
        },
        { palette, theme },
      );
      await page.goto("/#/torrents");
      await expect(page.getByRole("article")).toHaveCount(2);
      await expect(page.locator("html")).toHaveAttribute(
        "data-palette",
        palette,
      );
      await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
      for (const destination of ["/", "/add", "/settings", "/torrents"]) {
        await page.goto(`/#${destination}`);
        await expect(page.locator("#main h1")).toBeVisible();
        await page.locator("#main").evaluate(async (element) => {
          await Promise.all(
            element
              .getAnimations({ subtree: true })
              .map((animation) => animation.finished.catch(() => {})),
          );
        });
        const results = await new AxeBuilder({ page })
          .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
          .analyze();
        await info.attach(
          `accessibility-${destination.replaceAll("/", "") || "overview"}.json`,
          {
            body: JSON.stringify(results),
            contentType: "application/json",
          },
        );
        expect(results.violations).toEqual([]);
      }
      const opener = page.getByRole("button", {
        name: "Quick actions",
        exact: true,
      });
      await opener.focus();
      await page.keyboard.press("Control+k");
      const dialog = page.getByRole("dialog");
      await expect(dialog).toBeVisible();
      await dialog.getByRole("textbox").fill("settings");
      await page.keyboard.press("Enter");
      await expect(
        page.getByRole("heading", { level: 1, name: "Settings", exact: true }),
      ).toBeVisible();
      await opener.focus();
      await page.keyboard.press("Control+k");
      await page.keyboard.press("Escape");
      await expect(opener).toBeFocused();
      await page.screenshot({
        path: info.outputPath(`${palette}-${theme}.png`),
        fullPage: true,
      });
    });
  }
for (const width of [320, 768, 2560])
  for (const scale of [100, 125, 150, 200]) {
    test(`adaptive navigation ${width}px ${scale}%`, async ({ page }) => {
      await upgradeFixture(page, 2);
      await page.setViewportSize({ width, height: 900 });
      await page.addInitScript((scale) => {
        localStorage.setItem("flow.ui.textScale", JSON.stringify(scale));
        localStorage.setItem("i18nextLng", "en");
      }, scale);
      for (const path of ["/", "/torrents", "/add", "/settings"]) {
        await page.goto(`/#${path}`);
        await expect(page.locator("#main h1")).toBeVisible();
        await expect
          .poll(() =>
            page.evaluate(() =>
              document.documentElement.scrollWidth <= innerWidth + 1
                ? []
                : Array.from(document.querySelectorAll("body *"))
                    .filter((e) => {
                      const r = e.getBoundingClientRect();
                      return r.width > 0 && r.right > innerWidth + 1;
                    })
                    .slice(0, 12)
                    .map((e) => ({
                      tag: e.tagName,
                      css: e.className,
                      text: e.textContent?.slice(0, 80),
                      width: e.getBoundingClientRect().width,
                      right: e.getBoundingClientRect().right,
                    })),
            ),
          )
          .toEqual([]);
      }
    });
  }
for (const language of ["en", "ru", "ua", "bg", "fr", "ro", "zh"]) {
  test(`localized phone layout ${language}`, async ({ page }, info) => {
    await upgradeFixture(page, 1);
    await page.setViewportSize({ width: 375, height: 812 });
    await page.addInitScript((language) => {
      localStorage.setItem("i18nextLng", language);
      localStorage.setItem("flow.ui.textScale", "200");
    }, language);
    const labels = JSON.parse(
      readFileSync(
        new URL(
          `../src/i18n/locales/${language}/translation.json`,
          import.meta.url,
        ),
        "utf8",
      ),
    );
    await page.goto("/#/torrents");
    await expect(page.getByRole("article")).toHaveCount(1);
    await page
      .getByRole("button", {
        name: labels.app.expressive.commands,
        exact: true,
      })
      .click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    expect(await dialog.evaluate((e) => e.scrollWidth <= e.clientWidth)).toBe(
      true,
    );
    await page.keyboard.press("Escape");
    await page.setViewportSize({ width: 915, height: 412 });
    await expect
      .poll(() =>
        page.evaluate(() =>
          document.documentElement.scrollWidth <= innerWidth + 1
            ? []
            : Array.from(document.querySelectorAll("body *"))
                .filter((e) => {
                  const r = e.getBoundingClientRect();
                  return r.width > 0 && r.right > innerWidth + 1;
                })
                .slice(0, 12)
                .map((e) => ({
                  tag: e.tagName,
                  css: e.className,
                  text: e.textContent?.slice(0, 80),
                  width: e.getBoundingClientRect().width,
                  right: e.getBoundingClientRect().right,
                })),
        ),
      )
      .toEqual([]);
    await page.screenshot({
      path: info.outputPath(`${language}-landscape.png`),
    });
  });
}
test("Recent retains truthful provenance and explicit resume handoff", async ({
  page,
}) => {
  await upgradeFixture(page, 1);
  const hash = "0".repeat(40);
  let removed = 0;
  await page.addInitScript(() => {
    localStorage.setItem("i18nextLng", "en");
    Object.defineProperty(navigator, "userAgent", { value: "Android" });
  });
  await page.route("**/flow/recent", (route) => {
    if (route.request().method() === "POST") {
      removed++;
      return route.fulfill({ json: { schema_version: 1, ok: true } });
    }
    return route.fulfill({
      json: {
        schema_version: 1,
        enabled: true,
        read_only: false,
        items: [
          {
            hash,
            file_index: 1,
            title: "Fixture",
            file_path: "Unicode.Эпизод.S01E01.mkv",
            last_opened_at: new Date().toISOString(),
            viewed: { timecode: 42.5, position_source: "legacy_saved" },
          },
        ],
      },
    });
  });
  await page.goto("/#/");
  await expect(
    page.getByText("Legacy saved position:", { exact: false }),
  ).toBeVisible();
  await expect(
    page.getByText("Explicitly marked complete", { exact: true }),
  ).toHaveCount(0);
  const link = page.getByRole("link", {
    name: "Play in Just Player",
    exact: true,
  });
  await expect(link).not.toHaveAttribute("href", /i.position=/);
  await page.getByRole("button", { name: /Use saved position/ }).click();
  await expect(link).toHaveAttribute("href", /i.position=42500;/);
  await page
    .getByRole("button", { name: "Clear Recent history", exact: true })
    .click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  expect(removed).toBe(0);
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(removed).toBe(0);
});
test("reduced motion and forced colors preserve focus", async ({ page }) => {
  await upgradeFixture(page, 1);
  await page.emulateMedia({
    reducedMotion: "reduce",
    forcedColors: "active",
    contrast: "more",
  });
  await page.goto("/#/torrents");
  await expect(page.getByRole("article")).toHaveCount(1);
  await page.keyboard.press("Control+k");
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.keyboard.press("Tab");
  expect(
    await page.evaluate(
      () => !!document.activeElement?.closest('[role="dialog"]'),
    ),
  ).toBe(true);
  expect(
    await page
      .getByRole("dialog")
      .evaluate((e) => getComputedStyle(e).animationName),
  ).toBe("none");
});

test("command palette opens appearance repeatedly without closing it", async ({
  page,
}) => {
  await upgradeFixture(page, 1);
  await page.addInitScript(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("/#/torrents");
  for (let attempt = 0; attempt < 2; attempt++) {
    await expect(page.getByRole("article")).toHaveCount(1);
    await page
      .getByRole("button", { name: "Quick actions", exact: true })
      .focus();
    await page.keyboard.press("Control+k");
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("textbox").fill("appearance");
    await page.keyboard.press("Enter");
    await expect(dialog).toHaveCount(0);
    await expect(page.locator("[data-appearance]")).toHaveAttribute("open", "");
    await expect(page.locator("[data-appearance] summary")).toBeFocused();
  }
});

test("progressive files remain searchable and sidecars stay metadata-only", async ({
  page,
}) => {
  await upgradeFixture(page, 1);
  await page.addInitScript(() => localStorage.setItem("i18nextLng", "en"));
  const hash = "0".repeat(40);
  const filename = "Сериал.你好.S03E210.2160p.HDR.mkv";
  let lists = 0;
  let reads = 0;
  let minted = 0;
  await page.route("**/torrents", (route) =>
    route.fulfill({
      json: {
        hash,
        title: "Fixture 0",
        stat: 3,
        file_stats: Array.from({ length: 210 }, (_, i) => ({
          id: i + 1,
          path:
            i === 209
              ? filename
              : `Episode.S03E${String(i + 1).padStart(3, "0")}.mkv`,
          length: 10485760,
        })),
      },
    }),
  );
  await page.route("**/viewed", (route) => route.fulfill({ json: [] }));
  await page.route("**/flow/playback-link", (route) => {
    minted++;
    return route.fulfill({
      json: {
        path: "/flow/play/fixture-subtitle-claim/media.mkv",
        subtitle_path: "/flow/play/fixture-subtitle-claim/media.srt",
        expires_at: new Date(Date.now() + 300000).toISOString(),
      },
    });
  });
  await page.route(`**/subs/${hash}/210*`, (route) => {
    if (new URL(route.request().url()).searchParams.has("track")) reads++;
    else lists++;
    return route.fulfill({
      json: {
        schema_version: 1,
        source: "torrent_metadata",
        known: true,
        truncated: false,
        tracks: [
          {
            file_index: 211,
            path: filename.replace(".mkv", ".en.srt"),
            size: 100,
            format: "srt",
            language: "en",
            forced: false,
            hearing_impaired: false,
            match: "exact",
          },
        ],
      },
    });
  });
  await page.goto("/#/torrents");
  await page.getByRole("button", { name: "Files / Play", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.locator("article")).toHaveCount(100);
  expect(lists).toBe(0);
  expect(reads).toBe(0);
  await dialog
    .getByRole("textbox", { name: "Search", exact: true })
    .fill("你好");
  await expect(dialog.locator("article")).toHaveCount(1);
  await expect(
    dialog.getByText(filename, { exact: false }).first(),
  ).toBeVisible();
  await dialog.getByText("Torrent subtitles", { exact: true }).click();
  await expect.poll(() => lists).toBe(1);
  await dialog.locator("select").selectOption("211");
  expect(reads).toBe(0);
  await expect(
    dialog.getByRole("link", {
      name: "Download original subtitle",
      exact: true,
    }),
  ).toHaveAttribute(
    "href",
    /^\/flow\/play\/fixture-subtitle-claim\/media\.srt$/,
  );
  await expect.poll(() => minted).toBe(1);
  await dialog.getByRole("button", { name: /expiring link/i }).click();
  await expect.poll(() => minted).toBe(2);
});

for (const unavailable of [false, true]) {
  test(`Recent loading keeps layout stable (${unavailable ? "older" : "current"} server)`, async ({
    page,
  }) => {
    await upgradeFixture(page, 1);
    await page.addInitScript(() => localStorage.setItem("i18nextLng", "en"));
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route("**/flow/recent", async (route) => {
      await held;
      await route.fulfill(
        unavailable
          ? { status: 404, json: { error: "older server" } }
          : {
              json: {
                schema_version: 1,
                enabled: true,
                read_only: false,
                items: [],
              },
            },
      );
    });
    try {
      await page.goto("/#/");
      const heading = page.getByRole("heading", {
        name: "Recently opened",
        exact: true,
      });
      await expect(heading).toBeVisible();
      await page.locator("#main").evaluate(async (element) => {
        await Promise.all(
          element
            .getAnimations({ subtree: true })
            .map((animation) => animation.finished.catch(() => {})),
        );
      });
      const below = () =>
        heading.evaluate(
          (element) =>
            element
              .closest("section")!
              .nextElementSibling!.getBoundingClientRect().y,
        );
      const before = await below();
      release();
      await expect(
        page.getByText(
          unavailable
            ? /Recent history is unavailable/
            : "No recent video deliveries.",
        ),
      ).toBeVisible();
      await expect
        .poll(async () => Math.abs((await below()) - before))
        .toBeLessThanOrEqual(1);
    } finally {
      release();
    }
  });
}
