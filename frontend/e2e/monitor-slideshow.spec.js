import { test, expect } from '@playwright/test';

test('Monitor-Slideshow: lädt ohne Login und zeigt Elemente', async ({ page }) => {
    await page.goto('/monitor');
    
    // Smoke check: the page should load and not redirect to /login
    await expect(page).toHaveURL(/\/monitor/);
    
    // The monitor might fetch GET /api/monitor/slides
    // Verify there are no critical errors on screen
    const bodyText = await page.locator('body').textContent();
    // Ensure it's not a 404 or raw JSON.
    expect(bodyText).not.toContain('Not Found');
});
