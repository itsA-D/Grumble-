import { test, expect } from '@playwright/test';

// The core loop from PRD section 6 M1: submit, see it in the feed, open the
// detail page, vote, and render a hub from server HTML.

// Alphabetic so the PII detector does not read a timestamp as a phone number.
function token(): string {
  return Array.from({ length: 6 }, () => String.fromCharCode(97 + Math.floor(Math.random() * 26))).join('');
}

test('a submitted frustration publishes, opens its detail page, and appears in the Latest feed', async ({
  page,
}) => {
  const body = `The library on my street closes every lunchtime and the queue outside is absurd ${token()}`;

  await page.goto('/submit');
  await page.getByLabel('What is the frustration?').fill(body);
  await page.getByLabel('Category').selectOption('tech');
  await page.getByLabel('Country').selectOption('GB');
  await page.getByRole('button', { name: 'Post frustration' }).click();

  // A published submission redirects to its own detail page.
  await expect(page).toHaveURL(/\/p\//);
  await expect(page.getByRole('heading', { level: 1 })).toContainText('library on my street');

  // And it shows up on the Latest tab of the feed.
  await page.goto('/?tab=latest');
  await expect(page.getByText('library on my street', { exact: false }).first()).toBeVisible();
});

test('voting toggles over HTMX and flips the button state', async ({ page }) => {
  await page.goto('/?tab=latest');
  await page.locator('article').first().getByRole('link').first().click();
  await expect(page).toHaveURL(/\/p\//);

  const vote = page.getByRole('button', { name: /vote/i });
  await expect(vote).toHaveAttribute('aria-pressed', 'false');

  await vote.click();
  await expect(page.getByRole('button', { name: /vote/i })).toHaveAttribute('aria-pressed', 'true');

  await page.getByRole('button', { name: /vote/i }).click();
  await expect(page.getByRole('button', { name: /vote/i })).toHaveAttribute('aria-pressed', 'false');
});

test('a category hub renders server-side HTML with pain cards', async ({ page }) => {
  const resp = await page.goto('/c/tech');
  expect(resp?.status()).toBe(200);

  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  await expect(page.locator('article').first()).toBeVisible();

  // The cards are in the initial HTML response, not injected by the client.
  const html = await resp!.text();
  expect(html).toContain('<article');
});
