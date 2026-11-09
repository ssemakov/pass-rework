import { expect, type Page } from "@playwright/test";
export const master = "public test master O'Connor 🍋";
export async function command(page: Page, name: string) {
  await expect(page.locator(".app")).toHaveAttribute("data-ready", "true");
  await page.evaluate(
    (name) =>
      window.dispatchEvent(new CustomEvent("pass-command", { detail: name })),
    name,
  );
}
export async function create(page: Page) {
  await command(page, "create");
  await page
    .getByRole("button", { name: "Select storage path", exact: true })
    .click();
  await page
    .getByPlaceholder("Enter your storage Master Password")
    .fill(master);
  await page
    .getByPlaceholder("Enter your storage Master Password")
    .press("Enter");
  await page
    .getByPlaceholder("Please confirm the Master Password")
    .fill(master);
  await page
    .getByPlaceholder("Please confirm the Master Password")
    .press("Enter");
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Add new secret", exact: true }),
  ).toBeEnabled();
}
export async function open(page: Page, key = master) {
  await command(page, "open");
  await page
    .getByPlaceholder("Enter your Master Password", { exact: true })
    .fill(key);
  await page
    .getByPlaceholder("Enter your Master Password", { exact: true })
    .press("Enter");
}
export async function add(page: Page, name: string, password: string) {
  await page
    .getByRole("button", { name: "Add new secret", exact: true })
    .click();
  await page.getByLabel("Item name", { exact: true }).fill(name);
  await page.getByLabel("Username", { exact: true }).fill("user@example.test");
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page
    .getByLabel("Website URL", { exact: true })
    .fill("https://example.test");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByLabel("Search stored secrets")).toBeVisible();
  await expect(page.getByLabel("Search stored secrets")).toHaveValue("");
  await page.getByLabel("Search stored secrets").fill(name);
  await expect(page.getByRole("article", { name, exact: true })).toBeVisible();
}
export async function lock(page: Page) {
  await page.keyboard.press("Meta+Shift+Backquote");
  await expect(
    page.getByPlaceholder("Enter your Master Password", { exact: true }),
  ).toBeVisible();
}
