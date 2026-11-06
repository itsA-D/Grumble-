import { test, expect } from '@playwright/test';

// Security behaviours from the Phase 1 gate: the self-harm safety floor, output
// escaping, and a strict Content Security Policy that leaves HTMX working.

// Alphabetic so the PII detector does not read a timestamp as a phone number.
function token(): string {
  return Array.from({ length: 6 }, () => String.fromCharCode(97 + Math.floor(Math.random() * 26))).join('');
}

test('a self-harm submission is held and never appears publicly', async ({ page }) => {
  const body = `Some mornings I harm myself just thinking about the commute ${token()}`;

  await page.goto('/submit');
  await page.getByLabel('What is the frustration?').fill(body);
  await page.getByLabel('Category').selectOption('health');
  await page.getByLabel('Country').selectOption('GB');
  await page.getByRole('button', { name: 'Post frustration' }).click();

  // Held for review: no detail page, just the acknowledgement.
  await expect(page.getByText('being reviewed', { exact: false })).toBeVisible();
  await expect(page).not.toHaveURL(/\/p\//);

  // It is absent from the public feed.
  await page.goto('/?tab=latest');
  await expect(page.getByText('harm myself', { exact: false })).toHaveCount(0);
});

test('user HTML in a body is escaped, never executed', async ({ page }) => {
  let alerted = false;
  page.on('dialog', async (dialog) => {
    alerted = true;
    await dialog.dismiss();
  });

  const marker = `xssmarker${token()}`;
  const body = `<script>alert(1)</script> ${marker} the escalator is always broken`;

  await page.goto('/submit');
  await page.getByLabel('What is the frustration?').fill(body);
  await page.getByLabel('Category').selectOption('other');
  await page.getByLabel('Country').selectOption('US');
  await page.getByRole('button', { name: 'Post frustration' }).click();

  await page.goto('/?tab=latest');
  await expect(page.getByText(marker)).toBeVisible();

  expect(alerted, 'no dialog should fire from user text').toBe(false);
  expect(await page.locator('script:has-text("alert(1)")').count()).toBe(0);
});

test('pages load, forms submit, and HTMX votes run with no CSP violations', async ({ page }) => {
  const violations: string[] = [];
  page.on('console', (msg) => {
    const text = msg.text();
    if (/Content Security Policy|Refused to/i.test(text)) violations.push(text);
  });
  page.on('pageerror', (err) => violations.push(`pageerror: ${err.message}`));

  await page.goto('/');
  await page.goto('/submit');
  await page.goto('/?tab=latest');
  await page.goto('/c/tech');
  await page.goto('/country/US');
  await page.goto('/does-not-exist');

  // HTMX is loaded and functional under the CSP.
  const htmxType = await page.evaluate(() => typeof (window as { htmx?: unknown }).htmx);
  expect(htmxType).toBe('object');

  await page.goto('/?tab=latest');
  await page.locator('article').first().getByRole('link').first().click();
  await page.getByRole('button', { name: /vote/i }).click();
  await expect(page.getByRole('button', { name: /vote/i })).toBeVisible();

  // The map runs its own external script and must stay CSP-clean too.
  await page.goto('/map');
  await expect(page.locator('#map-svg')).toBeVisible();
  await page.goto('/map?category=housing');
  await expect(page.locator('#map-svg')).toBeVisible();

  expect(violations).toEqual([]);
});
