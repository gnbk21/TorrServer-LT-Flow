import { test, expect } from "@playwright/test";
import { upgradeFixture } from "./upgrade-fixture";
import { readFileSync } from "node:fs";

for (const language of ["en", "ru", "ua", "bg", "fr", "ro", "zh"])
  for (const [width, height] of [
    [375, 812],
    [412, 915],
    [768, 1024],
    [1440, 900],
    [2560, 1440],
    [812, 375],
  ])
    test(`upgrade layout ${language} ${width}x${height}`, async ({
      page,
    }, info) => {
      await upgradeFixture(page);
      await page.setViewportSize({ width, height });
      const theme = width % 2 ? "dark" : "light";
      await page.addInitScript(
        ({ language, theme }) => {
          localStorage.setItem("i18nextLng", language);
          localStorage.setItem("flow.ui.theme", JSON.stringify(theme));
        },
        { language, theme },
      );
      const labels = JSON.parse(
        readFileSync(
          new URL(
            `../src/i18n/locales/${language}/translation.json`,
            import.meta.url,
          ),
          "utf8",
        ),
      ).nav;
      await page.goto("/");
      for (const path of ["/", "/torrents", "/settings", "/add"]) {
        if (path !== "/")
          await page.locator(`a[href="#${path}"]:visible`).first().click();
        await expect(
          page.getByRole("heading", {
            level: 1,
            name: labels[path === "/" ? "dashboard" : path.slice(1)],
            exact: true,
          }),
        ).toBeVisible();
        await expect(page.locator("html")).toHaveAttribute(
          "lang",
          language === "ua" ? "uk" : language,
        );
        await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
        expect(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth,
          ),
        ).toBe(true);
        const undersized = await page
          .locator("button:visible,input:visible,select:visible")
          .evaluateAll((elements) =>
            elements
              .filter(
                (element) => element.getBoundingClientRect().height < 43.9,
              )
              .map((element) => element.tagName),
          );
        expect(undersized).toEqual([]);
        if (language === "en" && [375, 1440].includes(width))
          await page.screenshot({
            path: info.outputPath(
              `${path.replaceAll("/", "") || "dashboard"}-${theme}.png`,
            ),
            fullPage: true,
          });
      }
      await page.keyboard.press("Tab");
      expect(
        await page.evaluate(() => document.activeElement?.tagName !== "BODY"),
      ).toBe(true);
    });

for (const [width, height] of [
  [375, 812],
  [812, 375],
]) {
  test(`mobile settings actions stay above navigation ${width}x${height}`, async ({
    page,
  }) => {
    await upgradeFixture(page);
    await page.setViewportSize({ width, height });
    await page.addInitScript(() => localStorage.setItem("i18nextLng", "en"));
    await page.goto("/#/settings");
    await page.getByRole("tab", { name: "Flow", exact: true }).click();
    await page
      .getByRole("checkbox", { name: /^Show advanced settings/ })
      .check();
    await expect(page.locator("#Flow__SwarmProfile")).toBeVisible();
    await page.evaluate(() => window.scrollTo(0, 0));
    const actions = page.locator(".actions.sticky");
    const navigation = page.locator("nav.lg\\:hidden");
    await expect
      .poll(async () => {
        const actionBounds = await actions.boundingBox();
        const navBounds = await navigation.boundingBox();
        return (
          !!actionBounds &&
          !!navBounds &&
          actionBounds.y + actionBounds.height <= navBounds.y
        );
      })
      .toBe(true);
    const submit = page.getByRole("button", {
      name: "Apply settings",
      exact: true,
    });
    await expect(submit).toHaveAttribute("form", "flow-settings-form");
    expect(
      await submit.evaluate(
        (element) => (element as HTMLButtonElement).form?.id,
      ),
    ).toBe("flow-settings-form");
    await page.locator("#Flow__SwarmProfile").selectOption("adaptive");
    await expect(submit).toBeEnabled();
    await submit.click();
    await expect(page.getByRole("dialog")).toBeVisible();
  });
}
