// Gemeinsame Helfer für die E2E-Smoke-Flows.

import { execSync } from 'node:child_process';

export const ADMIN_EMAIL = 'pflasch@philipp-reis-schule.de';
// Mock-IMAP (IMAP_HOST=mock im lokalen Stack) akzeptiert jedes Passwort.
export const ADMIN_PASSWORD = 'e2e-egal';

/**
 * Loggt über die echte Login-UI ein und wartet, bis die App steht.
 * (Die SPA macht beim Boot keinen Session-Restore — ein reines API-Login
 * ließe den Login-Screen stehen. Der Cookie-Jar wird mit page.request geteilt,
 * API-Seeding nach diesem Login ist also authentifiziert.)
 * @param {import('@playwright/test').Page} page
 * @param {string} [asEmail] anderer Benutzer (Mock-IMAP akzeptiert jedes Passwort)
 */
export async function uiLogin(page, asEmail = ADMIN_EMAIL) {
    await page.goto('/');

    const email = page.locator('#login-email');
    const password = page.locator('#login-password');

    // Erst fokussieren, dann füllen, dann verifizieren — die Svelte-Bindings
    // rendern kurz nach dem Mount; ungeduldiges fill landet sonst im falschen Feld.
    await email.click();
    await email.fill(asEmail);
    await password.click();
    await password.fill(ADMIN_PASSWORD);

    const { expect } = await import('@playwright/test');
    await expect(email).toHaveValue(asEmail);
    await expect(password).toHaveValue(ADMIN_PASSWORD);

    await page.getByRole('button', { name: 'Anmelden' }).click();
    await page.getByRole('button', { name: 'Abmelden' }).waitFor();
}

/**
 * Holt das CSRF-Token (Double-Submit-Cookie-Pattern) für schreibende API-Calls.
 * @param {import('@playwright/test').Page} page
 */
export async function csrfToken(page) {
    const res = await page.request.get('/api/csrf-token');
    const body = await res.json();
    return body.csrf_token;
}

/**
 * Schreibender API-Call mit CSRF-Header — für Test-Seeding (Schüler, Lieferanten …).
 * @param {import('@playwright/test').Page} page
 * @param {string} url
 * @param {object} data
 */
export async function apiPost(page, url, data) {
    const token = await csrfToken(page);
    return page.request.post(url, {
        data,
        headers: { 'X-CSRF-Token': token },
    });
}

/**
 * Schreibender API-Call (PATCH) mit CSRF-Header.
 * @param {import('@playwright/test').Page} page
 * @param {string} url
 * @param {object} data
 */
export async function apiPatch(page, url, data) {
    const token = await csrfToken(page);
    return page.request.patch(url, {
        data,
        headers: { 'X-CSRF-Token': token },
    });
}

/**
 * Seedet Testdaten direkt in die lokale Stack-DB (docker-compose.local.yml).
 * Nur für Entitäten ohne einfachen API-Weg (z. B. Buch-Exemplare).
 * @param {string} sql
 */
export function seedSQL(sql) {
    const container = process.env.E2E_DB_CONTAINER || 'bibliothek-db-local';
    execSync(`docker exec -i ${container} psql -U postgres -d bibliothek -v ON_ERROR_STOP=1`, {
        input: sql,
    });
}

/** Eindeutiger Suffix, damit Läufe auf derselben DB nicht kollidieren. */
export function uniqueSuffix() {
    return `${Date.now().toString(36)}${Math.floor(Math.random() * 1000)}`;
}
