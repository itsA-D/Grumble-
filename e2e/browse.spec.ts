import { test, expect } from '@playwright/test';

// Trending windows swap in place and keep the shell; saved frustrations survive
// a reload and can be removed with an announced count.

test('trending windows swap without losing the shell', async ({ page }) => {
  await page.goto('/trending');
  await expect(page.getByRole('heading', { level: 1, name: 'Trending frustrations' })).toBeVisible();
  await expect(page.locator('#trending-main')).toBeVisible();

  await page.getByRole('link', { name: 'Today' }).click();
  await expect(page).toHaveURL(/\/trending\?window=today/);
  await expect(page.locator('#trending-main a[aria-current="page"]')).toHaveText('Today');
  await expect(page.locator('nav[aria-label="Main"]')).toHaveCount(1);
  await expect(page.locator('#trending-main')).toBeVisible();

  await page.reload();
  await expect(page.locator('#trending-main a[aria-current="page"]')).toHaveText('Today');
  await expect(page.locator('#trending-main')).toBeVisible();
});

test('a saved pain persists and can be removed from the saved page', async ({ page }) => {
  await page.goto('/?tab=latest');
  const first = page.locator('article').first();
  const href = await first.getByRole('link').first().getAttribute('href');
  expect(href).toBeTruthy();

  await first.getByRole('button', { name: 'SAVE' }).click();
  await expect(first.getByRole('button', { name: 'SAVED' })).toHaveAttribute('aria-pressed', 'true');

  const saved = await page.goto('/saved');
  expect(saved?.status()).toBe(200);
  expect(await saved?.headerValue('x-robots-tag')).toBe('noindex');
  await expect(page.getByRole('heading', { level: 1, name: 'Saved' })).toBeVisible();

  const row = page.locator('#saved-list a[href="' + href + '"]').first();
  await expect(row).toBeVisible();

  const article = page.locator('#saved-list article').filter({ has: page.locator('a[href="' + href + '"]') });
  await article.getByRole('button', { name: 'REMOVE' }).click();
  await expect(page.locator('#saved-list a[href="' + href + '"]')).toHaveCount(0);
  await expect(page.locator('#saved-count')).toContainText('SAVED');

  await page.reload();
  await expect(page.locator('#saved-list a[href="' + href + '"]')).toHaveCount(0);
});
