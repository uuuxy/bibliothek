import { it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import Konten from '../benutzer/Konten.svelte';
import Logbuch from './Logbuch.svelte';
import VerwaltungsProtokoll from './VerwaltungsProtokoll.svelte';

// Die drei Listen hängen an vergebbaren Rechten (manage_users, audit_logs), nicht an der Rolle
// Administrator. Bei 403 stand dort „Nur für System-Administratoren", auch wenn eine andere
// Rolle das Recht bekommen hatte oder es einem Administrator genommen war. Jetzt steht da, was
// der Server sagt.
it.each([
	['Benutzerliste', Konten],
	['Logbuch', Logbuch],
	['Verwaltungsprotokoll', VerwaltungsProtokoll]
])('%s zeigt bei 403 die Meldung des Servers', async (_name, Ansicht) => {
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({
			ok: false,
			status: 403,
			headers: new Headers(),
			text: async () => JSON.stringify({ error: 'Keine Berechtigung für diese Aktion' })
		})
	);
	const screen = render(/** @type {any} */ (Ansicht));
	expect(await screen.findByText(/Keine Berechtigung für diese Aktion/)).toBeTruthy();
	expect(screen.queryByText(/Nur für System-Administratoren/)).toBeNull();
});
