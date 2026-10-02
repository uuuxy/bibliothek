import { test, expect } from '@playwright/test';
import { uiLogin, apiPost } from './helpers.js';
import { STUMM, istStumm, liesMailserver } from './mailserver.js';

// Während des Laufs zeigt die Mail-Einstellung des Stacks auf eine Adresse, an der nichts
// zuhört (global-setup.js, mailserver.js). Fehlt das, verschicken die Flows dieser Suite
// Mail über den Server, der im Stack eingetragen ist — am Entwicklungsrechner der der Schule.
test('Mailserver: im Lauf stumm, ein Versand endet an der toten Adresse', async ({ page }) => {
	// Erst die Einstellung, dann der Versand: Stünde sie nicht stumm, ginge die Probe hinaus.
	expect(
		istStumm(liesMailserver()),
		'die Mail-Einstellung steht nicht stumm — kein Versand zur Probe'
	).toBe(true);

	await uiLogin(page);
	const antwort = await apiPost(page, '/api/admin/settings/mail/test', {
		to: 'e2e-stumm@test.local'
	});

	// 502 mit der Adresse in der Meldung: Der Server hat genau dort angeklopft und nirgends sonst.
	expect(antwort.status(), 'Testversand gegen die tote Adresse').toBe(502);
	expect((await antwort.json()).error).toContain(`${STUMM.host}:${STUMM.port}`);
});
