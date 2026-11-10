import {
  test,
  expect,
  type Page,
  type APIRequestContext,
} from "@playwright/test";
import { master, command, add, lock } from "./helpers";

const loginField = (page: Page) =>
  page.getByPlaceholder("Enter your Master Password", { exact: true });
async function signup(page: Page) {
  const first = page.getByPlaceholder("Enter your storage Master Password", {
    exact: true,
  });
  await expect(first).toBeVisible();
  await first.fill(master);
  await first.press("Enter");
  const confirmation = page.getByPlaceholder(
    "Please confirm the Master Password",
    { exact: true },
  );
  await confirmation.fill(master);
  await confirmation.press("Enter");
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
}
async function login(page: Page, password = master) {
  await loginField(page).fill(password);
  await loginField(page).press("Enter");
}
async function stats(request: APIRequestContext) {
  return (await request.post("/test-stats")).json() as Promise<{
    picks: Record<string, number>;
    settings: string;
  }>;
}
async function restart(page: Page, request: APIRequestContext) {
  expect((await request.post("/test-restart")).status()).toBe(204);
  await page.reload();
  await expect(loginField(page)).toBeEnabled();
  await expect(loginField(page)).toHaveValue("");
}
test.beforeEach(async ({ page, request }) => {
  await request.post("/test-reset");
  await page.goto("/");
});

test("service startup failure is visible and recovers without showing an unconfigured login", async ({
  page,
  request,
}) => {
  await page.route("**/test-api/State", (route) =>
    route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({ error: "temporary test service failure" }),
    }),
  );
  await page.reload();
  await expect(page.getByRole("alert")).toHaveText(
    "Could not load storage. Reopen PassV2 to try again.",
  );
  await expect(loginField(page)).toHaveCount(0);
  await page.evaluate(() =>
    window.dispatchEvent(new CustomEvent("pass-command", { detail: "open" })),
  );
  expect((await stats(request)).picks).toEqual({});
  await page.unroute("**/test-api/State");
  await expect(
    page.getByPlaceholder("Enter your storage Master Password", {
      exact: true,
    }),
  ).toBeEnabled();
});

test("New Storage mismatch does not create; an existing default is never overwritten and Escape restores login", async ({
  page,
  request,
}) => {
  await signup(page);
  const before = (await stats(request)).settings;
  let creates = 0;
  page.on("request", (r) => {
    if (r.url().endsWith("/test-api/Create")) creates++;
  });
  await command(page, "create");
  const password = page.getByPlaceholder("Enter your storage Master Password", {
    exact: true,
  });
  await password.fill("different public master");
  await password.press("Enter");
  const confirmation = page.getByPlaceholder(
    "Please confirm the Master Password",
    { exact: true },
  );
  await confirmation.fill("mismatch");
  await confirmation.press("Enter");
  await expect(page.getByRole("alert")).toHaveText("Passwords don't match");
  expect(creates).toBe(0);
  await confirmation.fill("different public master");
  await confirmation.press("Enter");
  await expect(page.getByRole("alert")).toContainText(
    "A storage file already exists",
  );
  expect(creates).toBe(1);
  expect((await stats(request)).settings).toBe(before);
  await page.keyboard.press("Escape");
  await expect(loginField(page)).toHaveValue("");
  await login(page);
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
});

test("first launch creates storage without a picker; real restart restores path and idle preference", async ({
  page,
  request,
}) => {
  await expect(loginField(page)).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Import Storage…", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Open Storage…", exact: true }),
  ).toBeVisible();
  await signup(page);
  await add(page, "Restart example", "public persisted entry");
  await command(page, "settings");
  await page.getByLabel("Lock after inactivity").selectOption("15");
  await expect(page.getByLabel("Lock after inactivity")).toBeEnabled();
  await page.getByRole("button", { name: "Close", exact: true }).click();
  const saved = await stats(request);
  expect(saved.picks).toEqual({});
  expect(JSON.parse(saved.settings).default_storage_path).toMatch(
    /storage\.pass$/,
  );
  expect(JSON.parse(saved.settings).idle_minutes).toBe(15);
  expect(saved.settings).not.toContain(master);
  expect(saved.settings).not.toContain("public persisted entry");
  await restart(page, request);
  await expect(
    page.getByText("public persisted entry", { exact: true }),
  ).toHaveCount(0);
  await login(page);
  await page.getByLabel("Search stored secrets").fill("Restart");
  await expect(
    page.getByRole("article", { name: "Restart example", exact: true }),
  ).toBeVisible();
  await command(page, "settings");
  await expect(page.getByLabel("Lock after inactivity")).toHaveValue("15");
  expect((await stats(request)).picks).toEqual({});
});

test("rejected password performs one attempt, opens no dialog and permits a correct retry", async ({
  page,
  request,
}) => {
  await signup(page);
  await add(page, "Retry example", "public retry entry");
  await restart(page, request);
  const before = await stats(request);
  const calls: string[] = [];
  page.on("request", (r) => {
    const method = r.url().split("/test-api/")[1];
    if (method && method !== "State") calls.push(method);
  });
  let entered!: () => void, release!: () => void;
  const ready = new Promise<void>((r) => (entered = r)),
    blocked = new Promise<void>((r) => (release = r));
  await page.route("**/test-api/Unlock", async (route) => {
    const response = await route.fetch();
    entered();
    await blocked;
    await route.fulfill({ response });
  });
  await login(page, "wrong password");
  await ready;
  await page.evaluate(() => {
    for (let i = 0; i < 5; i++)
      document
        .querySelector(".auth-container form")!
        .dispatchEvent(
          new Event("submit", { bubbles: true, cancelable: true }),
        );
  });
  release();
  await expect(page.getByRole("alert")).toHaveText("Wrong password");
  await expect(loginField(page)).toBeEnabled();
  await expect(loginField(page)).toHaveValue("wrong password");
  await page.waitForTimeout(1100); // Observe beyond the status polling interval.
  expect(calls).toEqual(["Unlock"]);
  expect(await stats(request)).toEqual(before);
  await page.unroute("**/test-api/Unlock");
  await login(page);
  await page.getByLabel("Search stored secrets").fill("Retry");
  await expect(
    page.getByRole("article", { name: "Retry example", exact: true }),
  ).toBeVisible();
  await lock(page);
  await expect(loginField(page)).toHaveValue("");
  await expect(page.getByRole("alert")).toHaveCount(0);
  await login(page);
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
  expect((await stats(request)).picks).toEqual({});
});

test("cancelled Open, backup and import preserve the saved storage and clear old credentials", async ({
  page,
  request,
}) => {
  await signup(page);
  await add(page, "Cancel example", "public cancel secret");
  const before = (await stats(request)).settings;
  await command(page, "backup"); // A cancelled native backup picker is not an error.
  // The first backup is deliberately valid; cancelling the next one exercises that path.
  await expect(page.getByRole("status")).toContainText("verified");
  await request.post("/test-picker-cancel");
  await command(page, "backup");
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await lock(page);
  await loginField(page).fill("old rejected draft");
  await request.post("/test-picker-cancel");
  await command(page, "open");
  await expect(loginField(page)).toBeEnabled();
  await expect(loginField(page)).toHaveValue("");
  await expect(page.getByRole("alert")).toHaveCount(0);
  await command(page, "import");
  await page
    .getByLabel("Backup Master Password", { exact: true })
    .fill("discard me");
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(loginField(page)).toHaveValue("");
  expect((await stats(request)).settings).toBe(before);
  await login(page);
  await page.getByLabel("Search stored secrets").fill("Cancel");
  const card = page.getByRole("article", {
    name: "Cancel example",
    exact: true,
  });
  await card
    .getByRole("button", { name: "Delete secret", exact: true })
    .click();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Cancel", exact: true })
    .click();
  await expect(card).toBeVisible();
});

test("missing remembered file stays on login until explicit Open selects its moved path", async ({
  page,
  request,
}) => {
  await signup(page);
  await lock(page);
  const before = (await stats(request)).settings;
  expect((await request.post("/test-move-storage")).status()).toBe(204);
  await restart(page, request);
  await login(page);
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(loginField(page)).toBeEnabled();
  expect((await stats(request)).picks).toEqual({});
  expect((await stats(request)).settings).toBe(before);
  await command(page, "open");
  await expect(loginField(page)).toHaveValue("");
  await login(page);
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
  expect(
    JSON.parse((await stats(request)).settings).default_storage_path,
  ).toMatch(/\.moved$/);
  await restart(page, request);
  await login(page);
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
});

test("first-run import rejects a bad key without changing settings; verified destination survives restart", async ({
  page,
  request,
}) => {
  await page
    .getByRole("button", { name: "Import Storage…", exact: true })
    .click();
  await page
    .getByLabel("Backup Master Password", { exact: true })
    .fill("wrong backup password");
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
  await expect(page.getByRole("alert")).toHaveText("Wrong password");
  expect((await stats(request)).settings).toBe("");
  expect((await stats(request)).picks).toEqual({ source: 1, destination: 1 });
  await page
    .getByLabel("Backup Master Password", { exact: true })
    .fill("public-fixture-password");
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(page.getByRole("status")).toHaveText(/^Imported \d+ secrets$/);
  await expect(
    page.getByRole("form", { name: "Import Storage" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Open storage", exact: true }).click();
  await expect(loginField(page)).toHaveValue("");
  expect(
    JSON.parse((await stats(request)).settings).default_storage_path,
  ).toMatch(/imported-1\.pass$/);
  await login(page);
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
  await restart(page, request);
  await login(page);
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
  expect((await stats(request)).picks).toEqual({ source: 1, destination: 1 });
});

test("wrong current password during rotation leaves the unlocked vault and preferences intact", async ({
  page,
  request,
}) => {
  await signup(page);
  const before = await stats(request);
  await command(page, "rotate");
  const dialog = page.getByRole("dialog");
  await dialog
    .getByLabel("Current Master Password", { exact: true })
    .fill("wrong");
  await dialog
    .getByLabel("New Master Password", { exact: true })
    .fill("replacement");
  await dialog
    .getByLabel("Confirm Master Password", { exact: true })
    .fill("replacement");
  await dialog.getByRole("button", { name: "Save", exact: true }).click();
  await expect(dialog.getByRole("alert")).toHaveText("Wrong password");
  expect(await stats(request)).toEqual(before);
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await lock(page);
  await login(page);
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
});
