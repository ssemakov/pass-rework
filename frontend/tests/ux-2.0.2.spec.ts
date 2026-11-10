import {
  test,
  expect,
  type Page,
  type APIRequestContext,
} from "@playwright/test";
import { master, command, create, add, lock } from "./helpers";
const login = (p: Page) =>
  p.getByPlaceholder("Enter your Master Password", { exact: true });
async function stats(request: APIRequestContext) {
  return (await request.post("/test-stats")).json();
}
async function importFields(page: Page, oldPassword = master) {
  await page
    .getByLabel("Backup Master Password", { exact: true })
    .fill(oldPassword);
  await page
    .getByRole("button", { name: "Select encrypted backup", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Select fresh destination", exact: true })
    .click();
  await page.getByLabel("New Master Password", { exact: true }).fill(master);
  await page
    .getByLabel("Confirm Master Password", { exact: true })
    .fill(master);
}
test.beforeEach(async ({ page, request }) => {
  await request.post("/test-reset");
  await page.setViewportSize({ width: 904, height: 506 });
  await page.goto("/");
});

test("import is centered, shows complete paths and has automatic source preservation", async ({
  page,
}, info) => {
  await command(page, "import");
  await expect(
    page.getByRole("button", { name: "Preserve original backup", exact: true }),
  ).toHaveCount(0);
  await expect(page.getByText("Backup to", { exact: true })).toBeVisible();
  await expect(
    page.getByText("The original storage file will be kept unchanged.", {
      exact: true,
    }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Select encrypted backup", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Select fresh destination", exact: true })
    .click();
  const paths = await page
    .locator(".transfer-form .path-text")
    .allTextContents();
  expect(paths).toHaveLength(2);
  expect(paths.every((p) => p.startsWith("/"))).toBe(true);
  expect(paths[0]).toMatch(/\/legacy\.pass$/);
  expect(paths[1]).toMatch(/\/imported-1\.pass$/);
  const form = await page
    .getByRole("form", { name: "Import Storage" })
    .boundingBox();
  expect(Math.abs(form!.y + form!.height / 2 - 253)).toBeLessThan(2);
  await page.screenshot({
    path: info.outputPath("import-centered.png"),
    caret: "hide",
  });
});

test("creation and import validate both fields immediately and recover when either field matches", async ({
  page,
}) => {
  const password = page.getByPlaceholder("Enter your storage Master Password", {
    exact: true,
  });
  const confirm = page.getByPlaceholder("Please confirm the Master Password", {
    exact: true,
  });
  await expect(confirm).toBeVisible();
  await password.fill("first");
  await confirm.fill("second");
  await expect(page.getByRole("alert")).toHaveText("Passwords don't match");
  await expect(confirm).toHaveAttribute("aria-invalid", "true");
  await password.fill("second");
  await expect(page.getByRole("alert")).toHaveCount(0);
  await command(page, "import");
  await importFields(page, "public-fixture-password");
  const importConfirm = page.getByLabel("Confirm Master Password", {
    exact: true,
  });
  await importConfirm.fill("different");
  await expect(page.getByRole("alert")).toHaveText("Passwords don't match");
  await expect(
    page.getByRole("button", { name: "Import", exact: true }),
  ).toBeDisabled();
  await importConfirm.fill(master);
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Import", exact: true }),
  ).toBeEnabled();
});

test("V1 consent stays focused, carries no password and migrates only after agreement", async ({
  page,
  request,
}) => {
  await request.post("/test-open-legacy");
  await command(page, "open");
  const before = await stats(request);
  let unlocks = 0;
  page.on("request", (r) => {
    if (r.url().endsWith("/test-api/Unlock")) unlocks++;
  });
  await login(page).fill("wrong");
  await login(page).press("Enter");
  await expect(page.getByRole("alert")).toHaveText("Wrong password");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await login(page).fill("public-fixture-password");
  await login(page).press("Enter");
  const dialog = page.getByRole("dialog", { name: "V1 storage", exact: true });
  await expect(dialog).toBeVisible();
  await expect(
    dialog.getByRole("button", { name: "Close dialog", exact: true }),
  ).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(dialog).toHaveCount(0);
  expect(unlocks).toBe(2);
  await page
    .getByRole("button", { name: "Migrate storage", exact: true })
    .click();
  await expect(dialog).toBeVisible();
  await dialog
    .getByRole("button", { name: "Migrate storage", exact: true })
    .click();
  const form = page.getByRole("form", { name: "Import Storage" });
  await expect(form).toBeVisible();
  await expect(
    page.getByLabel("Backup Master Password", { exact: true }),
  ).toHaveValue("");
  await expect(page.locator(".transfer-form .path-text").first()).toContainText(
    "/legacy.pass",
  );
  const prepared = await stats(request);
  expect(prepared.picks).toEqual({ open: 1 });
  expect(prepared.legacySHA256).toBe(before.legacySHA256);
  await page
    .getByLabel("Backup Master Password", { exact: true })
    .fill("public-fixture-password");
  await page
    .getByRole("button", { name: "Select fresh destination", exact: true })
    .click();
  await page.getByLabel("New Master Password", { exact: true }).fill(master);
  await page
    .getByLabel("Confirm Master Password", { exact: true })
    .fill(master);
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(form.getByRole("status")).toHaveText("Imported 2 secrets");
  await expect(form).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Import", exact: true }),
  ).toHaveCount(0);
  const done = await stats(request);
  expect(done.legacySHA256).toBe(before.legacySHA256);
  await page.getByRole("button", { name: "Open storage", exact: true }).click();
  await expect(login(page)).toHaveValue("");
  expect((await stats(request)).picks).toEqual(done.picks);
  await expect(page.getByRole("status")).toHaveCount(0);
  await login(page).fill(master);
  await login(page).press("Enter");
  await expect(
    page.getByText("2 secrets stored", { exact: true }),
  ).toBeVisible();
});

test("missing file reports friendly text, full path and explicit Open/Import recovery", async ({
  page,
  request,
}) => {
  await create(page);
  await lock(page);
  await request.post("/test-move-storage");
  await login(page).fill(master);
  await login(page).press("Enter");
  await expect(page.getByRole("alert")).toHaveText("Storage file not found");
  await expect(
    page.getByRole("button", { name: "Open storage", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Import storage", exact: true }),
  ).toBeVisible();
  await expect(page.locator(".storage-recovery .path-text")).toContainText(
    "/storage.pass",
  );
  expect(
    (await page
      .locator(".storage-recovery .path-text")
      .textContent())!.startsWith("/"),
  ).toBe(true);
});

test("unopenable permissions offer Open and Import instead of chmod instructions", async ({
  page,
  request,
}) => {
  await create(page);
  await lock(page);
  await request.post("/test-permissions");
  await login(page).fill(master);
  await login(page).press("Enter");
  await expect(page.getByRole("alert")).toContainText(
    "permissions prevent opening",
  );
  await expect(page.getByRole("alert")).not.toContainText("chmod");
  await expect(
    page.getByRole("button", { name: "Open storage", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Import storage", exact: true })
    .click();
  await expect(
    page.getByRole("form", { name: "Import Storage" }),
  ).toBeVisible();
});

test("empty import reports Imported0 on the same screen and opens without a second picker", async ({
  page,
  request,
}) => {
  await create(page);
  await command(page, "backup");
  await expect(page.getByRole("status")).toContainText("verified");
  await lock(page);
  await command(page, "import");
  await importFields(page);
  await page.getByRole("button", { name: "Import", exact: true }).click();
  const form = page.getByRole("form", { name: "Import Storage" });
  await expect(form.getByRole("status")).toHaveText("Imported 0 secrets");
  await expect(form).toBeVisible();
  const before = await stats(request);
  await page.getByRole("button", { name: "Open storage", exact: true }).click();
  expect((await stats(request)).picks).toEqual(before.picks);
  await expect(page.getByRole("status")).toHaveCount(0);
  await login(page).fill(master);
  await login(page).press("Enter");
  await expect(
    page.getByText("0 secrets stored", { exact: true }),
  ).toBeVisible();
});

test("delete close tooltip fits inside the dialog; header lock is right aligned and scrubs secrets", async ({
  page,
}) => {
  await create(page);
  await add(page, "Tooltip example", "public-locked-secret");
  await page
    .getByRole("button", { name: "Reveal password", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Delete secret", exact: true })
    .click();
  const dialog = page.getByRole("dialog", {
    name: "Delete secret?",
    exact: true,
  });
  await dialog
    .getByRole("button", { name: "Close dialog", exact: true })
    .hover();
  const tip = dialog.getByRole("tooltip");
  await expect(tip).toBeVisible();
  const t = await tip.boundingBox(),
    d = await dialog.boundingBox();
  expect(t!.x).toBeGreaterThanOrEqual(d!.x);
  expect(t!.x + t!.width).toBeLessThanOrEqual(d!.x + d!.width);
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  const button = page.getByRole("button", {
    name: "Lock storage",
    exact: true,
  });
  const b = await button.boundingBox(),
    c = await page.locator(".secret-count").boundingBox(),
    search = await page.locator(".search").boundingBox();
  expect(b!.x).toBeGreaterThan(c!.x + c!.width);
  expect(Math.abs(b!.x + b!.width - search!.x - search!.width)).toBeLessThan(1);
  await button.click();
  await expect(login(page)).toBeEnabled();
  await expect(login(page)).toHaveValue("");
  await expect(
    page.getByText("public-locked-secret", { exact: true }),
  ).toHaveCount(0);
});

test("Lock rejects a delayed successful import result", async ({ page }) => {
  await command(page, "import");
  await importFields(page, "public-fixture-password");
  let entered!: () => void, release!: () => void;
  const ready = new Promise<void>((r) => (entered = r)),
    blocked = new Promise<void>((r) => (release = r));
  await page.route("**/test-api/Import", async (route) => {
    const response = await route.fetch();
    entered();
    await blocked;
    await route.fulfill({ response });
  });
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await ready;
  await lock(page);
  release();
  await expect(page.getByRole("form", { name: "Import Storage" })).toHaveCount(
    0,
  );
  await expect(page.getByText(/Imported \d+ secrets/)).toHaveCount(0);
  await expect(login(page)).toHaveValue("");
});

test("Lock rejects a delayed V1 consent offer", async ({ page, request }) => {
  await request.post("/test-open-legacy");
  await command(page, "open");
  let entered!: () => void, release!: () => void;
  const ready = new Promise<void>((r) => (entered = r)),
    blocked = new Promise<void>((r) => (release = r));
  await page.route("**/test-api/Unlock", async (route) => {
    const response = await route.fetch();
    entered();
    await blocked;
    await route.fulfill({ response });
  });
  await login(page).fill("public-fixture-password");
  await login(page).press("Enter");
  await ready;
  await page.keyboard.press("Meta+Shift+Backquote");
  await expect(
    page.getByPlaceholder("Enter your storage Master Password", {
      exact: true,
    }),
  ).toBeEnabled();
  release();
  await expect(page.getByRole("dialog")).toHaveCount(0);
});
