import { test, expect, type Route } from "@playwright/test";
import { upgradeFixture } from "./upgrade-fixture";

for (const width of [412, 1440]) {
  test(`keyboard dialog focus and reduced motion at ${width}px`, async ({
    page,
  }) => {
    await upgradeFixture(page, 1);
    await page.route("**/torrents", (route) =>
      route.fulfill({
        json: {
          hash: "0".repeat(40),
          title: "Fixture 0",
          category: "Fixtures",
          stat: 3,
        },
      }),
    );
    await page.setViewportSize({ width, height: 915 });
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.goto("/#/torrents");
    await expect(
      page.getByRole("heading", { name: "Fixture 0", exact: true }),
    ).toBeVisible();
    await page.keyboard.press("Tab");
    await expect(
      page.getByRole("link", { name: "Skip to content" }),
    ).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(page.locator("#main")).toBeFocused();
    const opener = page.getByRole("button", { name: "Add", exact: true });
    await opener.focus();
    await page.keyboard.press("ArrowRight");
    const style = await opener.evaluate((element) => ({
      outline: getComputedStyle(element).outlineWidth,
      transition: getComputedStyle(element).transitionDuration,
      animation: getComputedStyle(element).animationName,
    }));
    expect(style).toEqual({
      outline: "2px",
      transition: "0s",
      animation: "none",
    });
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    for (const key of [
      ...Array(14).fill("Tab"),
      ...Array(14).fill("Shift+Tab"),
    ]) {
      await page.keyboard.press(key);
      expect(
        await page.evaluate(
          () => !!document.activeElement?.closest('[role="dialog"]'),
        ),
      ).toBe(true);
    }
    await page.keyboard.press("Escape");
    await expect(dialog).not.toBeVisible();
    await expect(opener).toBeFocused();
    // The next opener must replace the saved focus target rather than reuse Add.
    const edit = page.getByRole("button", {
      name: "Edit metadata",
      exact: true,
    });
    await edit.focus();
    await page.keyboard.press("Enter");
    await expect(dialog).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(dialog).not.toBeVisible();
    await expect(edit).toBeFocused();
    await page.locator('a[href="#/settings"]:visible').first().click();
    await expect(
      page.getByRole("heading", { name: "Settings", level: 1, exact: true }),
    ).toBeVisible();
    await page.locator('a[href="#/torrents"]:visible').first().click();
    await expect(
      page.getByRole("heading", { name: "Fixture 0", exact: true }),
    ).toBeVisible();
    await opener.focus();
    await page.keyboard.press("Enter");
    await expect(dialog).toBeVisible();
    // Browser/history navigation can unmount a dialog and its library opener.
    await page.goBack();
    await expect(
      page.getByRole("heading", { name: "Settings", level: 1, exact: true }),
    ).toBeVisible();
    await expect(dialog).not.toBeVisible();
    await expect(page.locator("#main")).toBeFocused();
  });
}

test("leaving the library cancels an unfinished detail request", async ({
  page,
}) => {
  await upgradeFixture(page, 1);
  let held: Route | undefined;
  let cancelled = false;
  page.on("requestfailed", (request) => {
    if (new URL(request.url()).pathname === "/torrents") cancelled = true;
  });
  await page.route("**/torrents", (route) => {
    held = route;
  });
  await page.goto("/#/settings");
  await expect(
    page.getByRole("heading", { name: "Settings", level: 1, exact: true }),
  ).toBeVisible();
  await page.locator('a[href="#/torrents"]:visible').first().click();
  await expect(
    page.getByRole("heading", { name: "Fixture 0", exact: true }),
  ).toBeVisible();
  const edit = page.getByRole("button", { name: "Edit metadata", exact: true });
  await edit.click();
  await expect(edit).toBeDisabled();
  await expect.poll(() => !!held).toBe(true);
  await page.goBack();
  await expect(
    page.getByRole("heading", { name: "Settings", level: 1, exact: true }),
  ).toBeVisible();
  await expect.poll(() => cancelled).toBe(true);
  await expect(page.getByRole("dialog")).not.toBeVisible();
  // Release the owned intercepted fixture even though its browser consumer left.
  await held!.abort().catch(() => {});
});
