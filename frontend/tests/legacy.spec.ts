import { test, expect, type Page } from "@playwright/test";
import { readFile, writeFile } from "node:fs/promises";
import { create, add } from "./helpers";
async function geometry(page: Page, selectors: string[]) {
  return page.evaluate(
    (selectors) =>
      selectors.map((selector) => {
        const e = document.querySelector(selector)!;
        const r = e.getBoundingClientRect(),
          s = getComputedStyle(e);
        return {
          x: r.x,
          y: r.y,
          width: r.width,
          height: r.height,
          fontSize: s.fontSize,
          fontWeight: s.fontWeight,
          color: s.color,
        };
      }),
    selectors,
  );
}
test("legacy reference geometry, colors, copy and rendered pixels", async ({
  page,
  context,
  request,
}, testInfo) => {
  await request.post("/test-reset");
  await page.setViewportSize({ width: 904, height: 506 });
  await page.goto("/");
  const reference = await context.newPage();
  await reference.setViewportSize({ width: 904, height: 506 });
  const samples: unknown[] = [];
  async function compare(name: string, product: string[], legacy: string[]) {
    await reference.setContent(
      await readFile(
        new URL("./legacy-reference/" + name + ".html", import.meta.url),
        "utf8",
      ),
    );
    await page.mouse.move(0, 0);
    await reference.mouse.move(0, 0);
    const a = await geometry(page, product),
      b = await geometry(reference, legacy);

    for (let i = 0; i < a.length; i++) {
      for (const key of ["fontSize", "fontWeight", "color"] as const)
        expect(a[i][key], `${name} ${product[i]} ${key}`).toBe(b[i][key]);
      for (const key of ["x", "y", "width", "height"] as const)
        expect(
          Math.abs(a[i][key] - b[i][key]),
          `${name} ${product[i]} ${key}`,
        ).toBeLessThan(1);
    }
    const expected = await reference.screenshot({
        animations: "disabled",
        caret: "hide",
      }),
      actual = await page.screenshot({
        animations: "disabled",
        caret: "hide",
        path: testInfo.outputPath(name + ".png"),
      });
    const diff = await reference.evaluate(
      async ([a, b]) => {
        const load = (s: string) =>
          new Promise<HTMLImageElement>((r) => {
            const i = new Image();
            i.onload = () => r(i);
            i.src = "data:image/png;base64," + s;
          });
        const x = await load(a),
          y = await load(b),
          c = document.createElement("canvas");
        c.width = x.width;
        c.height = x.height;
        const ctx = c.getContext("2d")!;
        ctx.drawImage(x, 0, 0);
        const p = ctx.getImageData(0, 0, c.width, c.height).data;
        ctx.clearRect(0, 0, c.width, c.height);
        ctx.drawImage(y, 0, 0);
        const q = ctx.getImageData(0, 0, c.width, c.height).data;
        let changed = 0;
        for (let n = 0; n < p.length; n += 4)
          if (
            Math.max(
              Math.abs(p[n] - q[n]),
              Math.abs(p[n + 1] - q[n + 1]),
              Math.abs(p[n + 2] - q[n + 2]),
            ) > 32
          )
            changed++;
        return changed / (c.width * c.height);
      },
      [expected.toString("base64"), actual.toString("base64")],
    );
    await writeFile(testInfo.outputPath(name + "-legacy.png"), expected);
    samples.push({
      screen: name,
      actual: a,
      reference: b,
      changedPixelFraction: diff,
    });
    console.log(
      `${name}: legacy geometry/styles match; changed pixels ${diff}`,
    );
    expect(diff, `${name} visual difference`).toBeLessThan(0.0002);
  }
  await expect(
    page.getByPlaceholder("Enter your Master Password", { exact: true }),
  ).toBeVisible();
  await compare(
    "login",
    [".auth-container .outlined-field", ".auth-container .icon-button"],
    [".MuiOutlinedInput-root", ".MuiIconButton-root"],
  );
  await create(page);
  await add(page, "Example Mail", "public fixture password");
  await add(page, "Example Notes", "public second password");
  await page.getByLabel("Search stored secrets").fill("");
  await compare(
    "home",
    [".fab", ".secret-count", ".search .outlined-field"],
    [".MuiFab-root", ".MuiTypography-h6", ".MuiOutlinedInput-root"],
  );
  await page.getByLabel("Search stored secrets").fill("example");
  await compare(
    "results",
    [".item-card", ".item-name", ".item-label > span", ".item-value"],
    [
      ".MuiCard-root",
      ".MuiCardContent-root .MuiBox-root",
      ".MuiTypography-caption",
      ".MuiCardContent-root .MuiGrid-grid-xs-10",
    ],
  );
  const passwordRow = page
    .getByRole("article", { name: "Example Mail", exact: true })
    .locator(".show-field")
    .nth(1);
  await expect(passwordRow.locator(".field-controls")).toHaveCSS(
    "opacity",
    "0",
  );
  await passwordRow.hover();
  await expect(passwordRow.locator(".field-controls")).toHaveCSS(
    "opacity",
    "1",
  );
  await passwordRow
    .getByRole("button", { name: "Reveal password", exact: true })
    .hover();
  await expect(
    passwordRow.getByRole("tooltip").filter({ hasText: "reveal password" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Add new secret", exact: true })
    .click();
  await compare(
    "add",
    [
      ".entry-form .outlined-field",
      ".entry-form .button.outlined",
      ".entry-form .button.contained",
    ],
    [".MuiOutlinedInput-root", ".MuiButton-outlined", ".MuiButton-contained"],
  );
  await writeFile(
    testInfo.outputPath("legacy-comparison.json"),
    JSON.stringify(samples, null, 2) + "\n",
  );
  await reference.close();
});
