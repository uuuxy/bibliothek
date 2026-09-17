import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, waitFor, fireEvent } from '@testing-library/svelte';
import StudentDirectoryToolbar from './StudentDirectoryToolbar.svelte';

// Gate für den Jahrgangsfilter der Leserdatei (OFFEN.md 9.5, Protokoll 5 und 6).
//
// Das Sichtungsprotokoll des Medienzentrums vom 16.09.2026: „Die Schülerdatei verfügt über
// keine Sortier- oder Filteroption nach Klassen bzw. Jahrgängen" — und unter
// Anpassungswünschen: „Derzeit ist eine Auswahl nach Klassen möglich. Gemeint ist jedoch
// eine Auswahl nach Jahrgängen."
//
// Geprüft wird die Werkzeugleiste: dass es das Feld gibt, dass es die Jahrgänge des
// SERVERS anbietet (und keine selbst erfundene Liste 5–13) und dass „Alle Jahrgänge"
// darin steht — ohne den Rückweg wäre der Filter eine Einbahnstraße.
describe('Jahrgangsfilter in der Leserdatei', () => {
	/** @param {Partial<Record<string, any>>} extra */
	function zeichne(extra = {}) {
		return render(StudentDirectoryToolbar, {
			searchQuery: '',
			jahrgang: '',
			jahrgaenge: [5, 6, 11, 13],
			trefferzahl: 0,
			...extra
		});
	}

	beforeEach(() => vi.clearAllMocks());

	it('bietet ein Feld zum Filtern nach Jahrgang', () => {
		const screen = zeichne();
		expect(screen.getByLabelText('Nach Jahrgang filtern')).not.toBeNull();
	});

	it('zeigt genau die Jahrgänge, die der Server nennt — samt Oberstufe', async () => {
		const screen = zeichne();
		await fireEvent.click(screen.getByLabelText('Nach Jahrgang filtern'));

		const text = screen.container.textContent ?? '';
		for (const j of [5, 6, 11, 13]) {
			expect(text, `Jahrgang ${j} fehlt`).toContain(`Jahrgang ${j}`);
		}
		// Der Jahrgang 11 heißt an dieser Schule „ET" und hat keine führende Ziffer. Dass
		// er in der Liste steht, ist der Beweis, dass die Jahrgänge vom Server kommen und
		// nicht im Browser aus Klassennamen geraten werden.
		expect(text).not.toContain('Jahrgang 7');
	});

	it('lässt sich zurücksetzen — „Alle Jahrgänge" steht in der Liste', async () => {
		const screen = zeichne({ jahrgang: '5' });
		await fireEvent.click(screen.getByLabelText('Nach Jahrgang filtern'));

		expect(screen.container.textContent ?? '').toContain('Alle Jahrgänge');
	});

	it('meldet die Auswahl nach oben, damit der Server neu gefragt wird', async () => {
		const onsearch = vi.fn();
		const screen = zeichne({ onsearch });

		await fireEvent.click(screen.getByLabelText('Nach Jahrgang filtern'));
		await fireEvent.click(screen.getByText('Jahrgang 6'));

		await waitFor(() => expect(onsearch).toHaveBeenCalled());
	});

	it('ohne Jahrgänge vom Server bleibt die Datei benutzbar', () => {
		// Eine leere Liste ohne Fehler ist ein legitimer Zustand (keine Schüler erfasst).
		// Dann steht dort nur „Alle Jahrgänge" — und gesucht werden kann weiter.
		const screen = zeichne({ jahrgaenge: [] });
		expect(screen.getByLabelText('Nach Jahrgang filtern')).not.toBeNull();
		expect(screen.getByLabelText('Leser suchen')).not.toBeNull();
	});

	it('sagt es, wenn die Jahrgänge nicht geladen werden konnten', () => {
		// Leer heißt leer, ein Ladefehler heißt Ladefehler. Ohne diese Unterscheidung
		// stünde im Feld „Alle Jahrgänge", und das läse sich wie „diese Schule hat keine
		// Jahrgänge" — dieselbe Sorte stiller Fehlantwort, die das Kollegiums-Portal am
		// 17.09.2026 abgelegt hat.
		const screen = zeichne({ jahrgaenge: [], jahrgaengeFehler: true });
		const feld = screen.getByLabelText('Nach Jahrgang filtern');

		expect(screen.container.textContent ?? '').toContain('Jahrgänge nicht geladen');
		expect(feld.getAttribute('disabled') ?? feld.getAttribute('aria-disabled')).not.toBeNull();
		// Die Datei selbst bleibt benutzbar — der Filter fehlt, die Suche nicht.
		expect(screen.getByLabelText('Leser suchen')).not.toBeNull();
	});
});
