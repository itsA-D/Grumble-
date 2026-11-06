import { test, expect, type Page } from '@playwright/test';

// The shared layout contract: the main navigation is the sticky top bar on
// every full-page response, at every viewport. HTMX swaps must never replace
// the shell.

const ROUTES = [
  '/',
  '/?tab=latest',
  '/trending',
  '/saved',
  '/map',
  '/submit',
  '/p/password-resets-take-twenty-minutes-and-land-in',
  '/c/tech',
  '/country/US',
  '/does-not-exist',
];

const NAV = 'nav[aria-label="Main"]';

async function navTop(page: Page): Promise<number> {
  return page.evaluate((sel) => {
    const nav = document.querySelector(sel);
    return nav ? nav.getBoundingClientRect().top : Number.NaN;
  }, NAV);
}

async function assertNavAtTop(page: Page): Promise<void> {
  const nav = page.locator(NAV);
  await expect(nav).toBeVisible();

  const box = await nav.boundingBox();
  expect(box, 'navigation must have a box').not.toBeNull();
  expect(box!.y).toBeLessThanOrEqual(1);

  // The centre of the bar and each visible link must be hit by the bar itself,
  // not by page content layered above it.
  const centreHit = await page.evaluate((sel) => {
    const nav = document.querySelector(sel);
    if (!nav) return false;
    const r = nav.getBoundingClientRect();
    const el = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
    return !!el && nav.contains(el);
  }, NAV);
  expect(centreHit, 'nav centre is covered by page content').toBe(true);

  const overflow = await page.evaluate((sel) => {
    const nav = document.querySelector(sel);
    return nav ? nav.scrollWidth - nav.clientWidth : 0;
  }, NAV);
  expect(overflow, 'navigation content overflows horizontally').toBeLessThanOrEqual(1);

  const links = nav.locator('a:visible');
  for (let i = 0; i < (await links.count()); i++) {
    const box = await links.nth(i).boundingBox();
    if (!box) continue;
    const hit = await page.evaluate(
      ({ sel, x, y }) => {
        const nav = document.querySelector(sel);
        const el = document.elementFromPoint(x, y);
        return !!nav && !!el && nav.contains(el);
      },
      { sel: NAV, x: box.x + box.width / 2, y: box.y + box.height / 2 },
    );
    expect(hit, `nav link ${i} is covered by page content`).toBe(true);
  }
}

for (const width of [1280, 360]) {
  test.describe(`navigation at ${width}px`, () => {
    test.use({ viewport: { width, height: 900 } });

    test('is at the top of every public page', async ({ page }) => {
      for (const route of ROUTES) {
        await page.goto(route);
        await assertNavAtTop(page);
      }
    });

    test('stays sticky at the bottom of a long page', async ({ page }) => {
      await page.goto('/map');
      await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
      await expect.poll(() => navTop(page)).toBe(0);
      await assertNavAtTop(page);
    });

    test('survives map chip, country and refresh', async ({ page }) => {
      await page.goto('/map');
      await assertNavAtTop(page);

      await page.locator('a[href="/map?category=housing"]').click();
      await expect(page).toHaveURL(/\/map\?category=housing/);
      await assertNavAtTop(page);

      await page.locator('path[data-country="GB"]').click();
      await expect(page.locator('#map-side')).toContainText('United Kingdom');
      await assertNavAtTop(page);

      await page.reload();
      await assertNavAtTop(page);

      // A direct visit to a filtered, pushed URL still renders the shell.
      await page.goto('/map?category=housing');
      await assertNavAtTop(page);
    });
  });
}

test('the mobile menu exposes the same links without JavaScript', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 900 });
  await page.goto('/map');

  const nav = page.locator(NAV);
  await nav.locator('summary', { hasText: 'Menu' }).click();
  await expect(nav.getByRole('link', { name: 'Map' })).toBeVisible();
  await expect(nav.getByRole('link', { name: 'Explore' })).toBeVisible();
  await expect(nav.getByRole('link', { name: 'Trending' })).toBeVisible();
  await expect(nav.getByRole('link', { name: 'Saved' })).toBeVisible();
  await nav.locator('summary', { hasText: 'Menu' }).click();
  await assertNavAtTop(page);
});
