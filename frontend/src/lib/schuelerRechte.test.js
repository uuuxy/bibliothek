import { describe, it, expect } from 'vitest';
import { schuelerRechte, darfAuskunftUeber } from './schuelerRechte.js';

// Die DSGVO-Auskunft über einen Leser mit Zugangskonto verlangt am Server zusätzlich
// manage_users (api/dsgvo_auskunft.go, TestDsgvoAuskunft_KontoVerlangtKontenrecht). Die Akte
// bietet den Knopf nach derselben Regel an — sonst stünde er bei der Leitung da und endete
// mit 403.
describe('darfAuskunftUeber', () => {
	const leitung = { rolle: 'leitung', permissions: ['view_students', 'manage_students_admin'] };
	const mitKontenrecht = { ...leitung, permissions: [...leitung.permissions, 'manage_users'] };
	const kollegeMitKonto = { email: 'kora@schule.invalid' };
	const ohneKonto = { email: '' };

	it('ohne manage_users nur für Leser ohne Konto', () => {
		const rechte = schuelerRechte(leitung);
		expect(darfAuskunftUeber(rechte, ohneKonto)).toBe(true);
		expect(darfAuskunftUeber(rechte, kollegeMitKonto)).toBe(false);
	});

	it('mit manage_users auch für Leser mit Konto', () => {
		expect(darfAuskunftUeber(schuelerRechte(mitKontenrecht), kollegeMitKonto)).toBe(true);
	});

	it('ohne das Recht der Route nie', () => {
		const nurKonten = { rolle: 'mitarbeiter', permissions: ['manage_users'] };
		expect(darfAuskunftUeber(schuelerRechte(nurKonten), ohneKonto)).toBe(false);
	});
});
