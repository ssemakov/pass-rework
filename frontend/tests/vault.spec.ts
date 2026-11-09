import { test, expect } from "@playwright/test";
import { master, command, create, open, add, lock } from "./helpers";
test.beforeEach(async ({ page, request }) => {
  await request.post("/test-reset");
  await page.goto("/");
});
test("encrypted backend CRUD, unchanged password, backup, rotation and restore", async ({
  page,
}) => {
  await create(page);
  await add(page, "Email example", "public-entry-secret");
  let card = page.getByRole("article", { name: "Email example", exact: true });
  await card
    .getByRole("button", { name: "Reveal password", exact: true })
    .click();
  await expect(card.getByLabel("Revealed password")).toHaveText(
    "public-entry-secret",
  );
  await card
    .getByRole("button", { name: "Copy password", exact: true })
    .click();
  await expect(
    card.getByRole("button", { name: "Copy password", exact: true }),
  ).toBeEnabled();
  await card.getByRole("button", { name: "Edit secret", exact: true }).click();
  await page.getByLabel("Item name", { exact: true }).fill("Edited email");
  await expect(page.getByLabel("Replace password")).not.toBeChecked();
  await expect(page.getByLabel("Password", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page.getByLabel("Search stored secrets").fill("Edited email");
  card = page.getByRole("article", { name: "Edited email", exact: true });
  await card
    .getByRole("button", { name: "Reveal password", exact: true })
    .click();
  await expect(card.getByLabel("Revealed password")).toHaveText(
    "public-entry-secret",
  );
  await command(page, "backup");
  await expect(page.getByRole("status")).toContainText("verified");
  await command(page, "rotate");
  const dialog = page.getByRole("dialog");
  await dialog
    .getByLabel("Current Master Password", { exact: true })
    .fill(master);
  await dialog
    .getByLabel("New Master Password", { exact: true })
    .fill("replacement public master");
  await dialog
    .getByLabel("Confirm Master Password", { exact: true })
    .fill("replacement public master");
  await dialog.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("status")).toContainText(
    "Master Password changed",
  );
  await lock(page);
  await expect(page.getByText("public-entry-secret")).toHaveCount(0);
  await open(page);
  await expect(page.getByRole("alert")).toHaveText("That didn't work");
  await page
    .getByPlaceholder("Enter your Master Password", { exact: true })
    .fill("replacement public master");
  await page
    .getByPlaceholder("Enter your Master Password", { exact: true })
    .press("Enter");
  await page.getByLabel("Search stored secrets").fill("Edited email");
  await page
    .getByRole("article", { name: "Edited email", exact: true })
    .getByRole("button", { name: "Delete secret", exact: true })
    .click();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(
    page.getByText("0 secrets stored", { exact: true }),
  ).toBeVisible();
  await lock(page);
  await command(page, "import");
  await page.getByLabel("Backup Master Password", { exact: true }).fill(master);
  await page
    .getByRole("button", { name: "Select encrypted backup", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Select fresh destination", exact: true })
    .click();
  await page
    .getByLabel("New Master Password", { exact: true })
    .fill("restored public master");
  await page
    .getByLabel("Confirm Master Password", { exact: true })
    .fill("restored public master");
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(page.getByRole("status")).toHaveText("Verified 1 secrets");
});
test("cancel preserves the password and explicit blank replacement persists", async ({
  page,
  request,
}) => {
  await create(page);
  await add(page, "Blank replacement", "original public secret");
  let card = page.getByRole("article", {
    name: "Blank replacement",
    exact: true,
  });
  await card.getByRole("button", { name: "Edit secret", exact: true }).click();
  await page.getByLabel("Replace password").check();
  await page
    .getByLabel("Password", { exact: true })
    .fill("cancelled public secret");
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await card
    .getByRole("button", { name: "Reveal password", exact: true })
    .click();
  await expect(card.getByLabel("Revealed password")).toHaveText(
    "original public secret",
  );
  await card.getByRole("button", { name: "Edit secret", exact: true }).click();
  await page.getByLabel("Replace password").check();
  await expect(page.getByLabel("Password", { exact: true })).toHaveValue("");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(
    card.getByRole("button", { name: "Reveal password", exact: true }),
  ).toHaveCount(0);
  await expect(
    card.getByRole("button", { name: "Copy password", exact: true }),
  ).toHaveCount(0);
  await lock(page);
  await open(page);
  await page.getByLabel("Search stored secrets").fill("Blank replacement");
  await expect(
    card.getByRole("button", { name: "Reveal password", exact: true }),
  ).toHaveCount(0);
  const status = await (
    await request.post("/test-api/State", { data: [] })
  ).json();
  const rows = await (
    await request.post("/test-api/List", { data: [status.generation, ""] })
  ).json();
  expect(rows[0].hasPassword).toBe(false);
  const persisted = await (
    await request.post("/test-api/Get", {
      data: [status.generation, rows[0].id],
    })
  ).json();
  expect(persisted.password).toBe("");
});
test("historical migration and text-only rendering", async ({ page }) => {
  await command(page, "import");
  await page
    .getByLabel("Backup Master Password", { exact: true })
    .fill("public-fixture-password");
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
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(page.getByRole("status")).toHaveText("Verified 2 secrets");
  await create(page);
  await add(page, "<img src=x onerror=alert(1)>", "public-secret");
  await expect(page.locator(".item-card img")).toHaveCount(0);
});
test("late reveal cannot appear after changing search or locking", async ({
  page,
}) => {
  await create(page);
  await add(page, "Alpha example", "alpha-public-secret");
  await add(page, "Beta example", "beta-public-secret");
  await page.getByLabel("Search stored secrets").fill("Alpha");
  let release!: () => void, entered!: () => void;
  const blocked = new Promise<void>((r) => (release = r)),
    ready = new Promise<void>((r) => (entered = r));
  await page.route("**/test-api/Get", async (route) => {
    const response = await route.fetch();
    entered();
    await blocked;
    await route.fulfill({ response });
  });
  await page
    .getByRole("article", { name: "Alpha example" })
    .getByRole("button", { name: "Reveal password", exact: true })
    .click();
  await ready;
  await page.getByLabel("Search stored secrets").fill("Beta");
  release();
  await expect(page.getByText("alpha-public-secret")).toHaveCount(0);
  await expect(page.getByLabel("Password hidden")).toHaveText("*****");
  await page.unroute("**/test-api/Get");
  await page
    .getByRole("button", { name: "Reveal password", exact: true })
    .click();
  await expect(page.getByLabel("Revealed password")).toHaveText(
    "beta-public-secret",
  );
  await lock(page);
  await expect(page.getByText("beta-public-secret")).toHaveCount(0);
});
test("keyboard dialogs, legacy empty search and Command-tilde lock", async ({
  page,
  request,
}) => {
  await page.setViewportSize({ width: 904, height: 506 });
  await create(page);
  await expect(page.getByLabel("Search stored secrets")).toBeFocused();
  await expect(page.getByRole("article")).toHaveCount(0);
  await command(page, "settings");
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.keyboard.press("Tab");
  await page.keyboard.press("Shift+Tab");
  await expect(
    page.getByRole("dialog").getByRole("button", { name: "Close dialog" }),
  ).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.evaluate(() =>
    window.dispatchEvent(
      new KeyboardEvent("keydown", {
        key: "l",
        code: "KeyL",
        metaKey: true,
        bubbles: true,
      }),
    ),
  );
  expect(
    (await (await request.post("/test-api/State", { data: [] })).json()).locked,
  ).toBe(false);
  await page.screenshot({
    path: "test-results/legacy-home.png",
    fullPage: true,
  });
  await expect(page.locator("body")).toHaveJSProperty("scrollWidth", 904);
  await lock(page);
});
test("late old native lock notification cannot relock a newer authentication", async ({
  page,
  request,
}) => {
  await create(page);
  const previous = await (
    await request.post("/test-api/State", { data: [] })
  ).json();
  await lock(page);
  await open(page);
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
  await page.evaluate(
    (old) =>
      window.dispatchEvent(new CustomEvent("pass-test-lock", { detail: old })),
    previous.generation,
  );
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
});
test("delayed pre-authentication status cannot replace the new session", async ({
  page,
}) => {
  let release!: () => void, arrived!: () => void;
  const pending = new Promise<void>((r) => (release = r)),
    ready = new Promise<void>((r) => (arrived = r));
  let held = false;
  await page.route("**/test-api/State", async (route) => {
    const response = await route.fetch(),
      body = await response.json();
    if (!held && body.locked) {
      held = true;
      arrived();
      await pending;
    }
    await route.fulfill({ response });
  });
  await ready;
  await create(page);
  release();
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
});

test("a second lock cancels a pending menu-open continuation", async ({
  page,
}) => {
  await create(page);
  let release!: () => void, entered!: () => void;
  const blocked = new Promise<void>((r) => (release = r)),
    ready = new Promise<void>((r) => (entered = r));
  let first = true,
    chooseCalls = 0;
  page.on("request", (r) => {
    if (r.url().endsWith("/test-api/Choose")) chooseCalls++;
  });
  await page.route("**/test-api/Lock", async (route) => {
    const response = await route.fetch();
    if (first) {
      first = false;
      entered();
      await blocked;
    }
    await route.fulfill({ response });
  });
  await command(page, "open");
  await ready;
  await page.keyboard.press("Meta+Shift+Backquote");
  const before = chooseCalls;
  release();
  await expect(
    page.getByPlaceholder("Enter your Master Password", { exact: true }),
  ).toBeVisible();
  await page.waitForTimeout(300);
  expect(chooseCalls).toBe(before);
});

test("a newer Create waits for the old lock and ignores its late notification", async ({
  page,
  request,
}) => {
  await create(page);
  let release!: () => void, entered!: () => void;
  const blocked = new Promise<void>((r) => (release = r)),
    ready = new Promise<void>((r) => (entered = r));
  await page.route("**/test-api/Lock", async (route) => {
    const response = await route.fetch();
    entered();
    await blocked;
    await route.fulfill({ response });
  });
  await command(page, "open");
  await ready;
  const state = await (
    await request.post("/test-api/State", { data: [] })
  ).json();
  await page.evaluate(
    (g) =>
      window.dispatchEvent(new CustomEvent("pass-test-lock", { detail: g })),
    state.generation,
  );
  await command(page, "create");
  await expect(
    page.getByPlaceholder("Enter your storage Master Password"),
  ).toHaveCount(0);
  release();
  const field = page.getByPlaceholder("Enter your storage Master Password");
  await expect(field).toBeVisible();
  await field.fill("public newer draft");
  await page.evaluate(
    (g) =>
      window.dispatchEvent(new CustomEvent("pass-test-lock", { detail: g })),
    state.generation,
  );
  await expect(field).toHaveValue("public newer draft");
});

test("an extra native lock cancels Open before its notification arrives", async ({
  page,
  request,
}) => {
  await create(page);
  let release!: () => void, entered!: () => void;
  const blocked = new Promise<void>((r) => (release = r)),
    ready = new Promise<void>((r) => (entered = r));
  let chooses = 0;
  page.on("request", (r) => {
    if (r.url().endsWith("/test-api/Choose")) chooses++;
  });
  await page.route("**/test-api/Lock", async (route) => {
    const response = await route.fetch();
    entered();
    await blocked;
    await route.fulfill({ response });
  });
  await command(page, "open");
  await ready;
  await request.post("/test-api/Lock", { data: [] });
  const state = await (
    await request.post("/test-api/State", { data: [] })
  ).json();
  const before = chooses;
  release();
  await expect(
    page.getByPlaceholder("Enter your Master Password", { exact: true }),
  ).toBeEnabled();
  await page.waitForTimeout(200);
  expect(chooses).toBe(before);
  await page.evaluate(
    (g) =>
      window.dispatchEvent(new CustomEvent("pass-test-lock", { detail: g })),
    state.generation,
  );
  expect(chooses).toBe(before);
});
