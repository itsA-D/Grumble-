import { test, expect } from '@playwright/test';

// The map loop from FR-MAP-1: the server renders the world and its fills, a
// country click swaps the side panel over HTMX, the category chips refilter the
// fills, and a country with no data in that filter shows the empty state.

test('the map renders server-side and a country click opens the side panel', async ({ page }) => {
  const resp = await page.goto('/map');
  expect(resp?.status()).toBe(200);

  await expect(page.getByRole('heading', { level: 1, name: 'World Pain Map' })).toBeVisible();
  const html = await resp!.text();
  expect(html).toContain('id="map-svg"');
  expect(html).toContain('data-country="GB"');

  await page.locator('path[data-country="GB"]').click();
  await expect(page).toHaveURL(/\/map$/);

  const side = page.locator('#map-side');
  await expect(side.getByRole('heading', { name: 'United Kingdom' })).toBeVisible();
  await expect(side.getByRole('link', { name: 'OPEN COUNTRY' })).toBeVisible();

  await side.getByRole('link', { name: 'CLEAR' }).click();
  await expect(page.locator('#map-side')).toContainText('TOP COUNTRIES');
});

test('a category chip refilters the map fills', async ({ page }) => {
  await page.goto('/map');

  const readFills = () =>
    page.$$eval('path[data-country]', (paths) =>
      paths.map((p) => `${p.getAttribute('data-country')}:${p.getAttribute('class')}`).join('|'),
    );

  const before = await readFills();

  await page.locator('a[href="/map?category=housing"]').click();
  await expect(page).toHaveURL(/\/map\?category=housing/);
  await expect(page.locator('a[href="/map?category=housing"]')).toHaveClass(/chip-active/);

  await expect
    .poll(async () => readFills(), { message: 'map fills should change with the filter' })
    .not.toBe(before);
});

test('a country with no data in the filter shows the empty state', async ({ page }) => {
  await page.goto('/map/country/CA?category=housing');

  const side = page.locator('#map-side');
  await expect(side.getByRole('heading', { name: 'Canada' })).toBeVisible();
  await expect(side).toContainText('No signal from Canada yet.');
  await expect(side.getByRole('link', { name: 'Post a frustration' })).toBeVisible();
});
