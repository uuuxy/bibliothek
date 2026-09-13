import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// „Abmelden" muss beim Server ankommen, auch wenn die Seite sofort danach verschwindet.
//
// Anlass (CI 13.09.2026, auth.spec.js rot): Das Frontend schickte POST /api/auth/logout
// ohne auf die Antwort zu warten und zeigte sofort das Login. Der Reload drei
// Millisekunden später brach die Anfrage ab — der Server hat die Sitzung nie gesperrt,
// nach dem Reload war der Admin wieder angemeldet. An einem geteilten Theken-Rechner
// heißt das: Abmelden, Tab zu, und der Nächste öffnet die Anwendung als Vorgänger.
//
// Nachgestellt wird der Tab, der schließt, während die Anfrage unterwegs ist. Die
// Verzögerung liegt auf der Ebene des Browser-Kontexts, damit sie das Schließen der Seite
// überlebt; geprüft wird danach über denselben Cookie-Speicher, ob die Sitzung tot ist.
test('Abmelden kommt beim Server an, auch wenn der Tab sofort schließt', async ({
	page,
	context
}) => {
	await uiLogin(page);

	const vorher = await context.request.get('/api/auth/me');
	expect(vorher.status()).toBe(200);

	await context.route('**/api/auth/logout', async (route) => {
		await new Promise((r) => setTimeout(r, 300));
		await route.continue().catch(() => {});
	});

	await page.getByRole('button', { name: 'Abmelden' }).click();
	await page.close();

	await expect
		.poll(async () => (await context.request.get('/api/auth/me')).status(), { timeout: 5000 })
		.toBe(401);
});
