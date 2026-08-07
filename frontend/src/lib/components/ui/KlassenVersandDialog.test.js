import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import KlassenVersandDialog from './KlassenVersandDialog.svelte';

/**
 * Zwei Fehler wären hier unsichtbar und teuer:
 *
 * 1. Nie zurückgesetzter UI-State: Der Dialog bleibt gemountet. Trägt er Abwahl und
 *    Fremdadresse in den nächsten Mahnlauf mit, verschickt der zweite Lauf still an
 *    jemand anderen als angezeigt.
 * 2. Eine vertippte Override-Adresse schickt den kompletten Lauf ins Leere — ohne
 *    Fehlermeldung, denn technisch ist der Versand ja „erfolgreich".
 */
const klassen = [
	{ klasse: '05A', lehrer_email: 'a@schule.de', schueler: [{}, {}] },
	{ klasse: '06B', lehrer_email: 'b@schule.de', schueler: [{}] },
	{ klasse: '07C', lehrer_email: '', schueler: [{}] }
];

// Die Wörter kommen jetzt von aussen — hier die des Mahnlaufs, damit die Tests
// weiter den echten Aufrufer abbilden.
const texte = {
	titel: 'Mahnlauf konfigurieren',
	beschreibung: 'Wähle die Klassen aus, für die Mahnungen generiert werden sollen.',
	aktion: 'anmahnen',
	hinweis: 'Bleibt das Feld leer, gehen die Mahnungen an die regulären Klassenleitungen.'
};

const senden = (getByRole) => getByRole('button', { name: /anmahnen/ });
const beschriftung = (el) => (el.textContent || '').replace(/\s+/g, ' ').trim();

describe('KlassenVersandDialog', () => {
	it('wählt beim Öffnen alle Klassen vor', () => {
		const { getByRole } = render(KlassenVersandDialog, {
			...texte,
			open: true,
			klassen,
			onclose: () => {},
			onconfirm: () => {}
		});
		expect(beschriftung(senden(getByRole))).toBe('3 Klassen anmahnen');
	});

	it('übergibt nur die angehakten Klassen und die getrimmte Override-Adresse', async () => {
		const onconfirm = vi.fn();
		const { getByRole, getByLabelText } = render(KlassenVersandDialog, {
			...texte,
			open: true,
			klassen,
			onclose: () => {},
			onconfirm
		});

		await fireEvent.click(getByRole('checkbox', { name: /06B/ }));
		await fireEvent.input(getByLabelText(/Alternative Empfänger/), {
			target: { value: '  sekretariat@schule.de  ' }
		});
		await fireEvent.click(senden(getByRole));

		expect(onconfirm).toHaveBeenCalledWith({
			klassen: ['05A', '07C'],
			overrideEmail: 'sekretariat@schule.de'
		});
	});

	it('sperrt den Versand bei ungültiger Adresse und ohne ausgewählte Klasse', async () => {
		const { getByRole, getByLabelText } = render(KlassenVersandDialog, {
			...texte,
			open: true,
			klassen,
			onclose: () => {},
			onconfirm: () => {}
		});

		await fireEvent.input(getByLabelText(/Alternative Empfänger/), {
			target: { value: 'sekretariat@' }
		});
		expect(senden(getByRole).disabled).toBe(true);

		await fireEvent.input(getByLabelText(/Alternative Empfänger/), { target: { value: '' } });
		for (const k of klassen) {
			await fireEvent.click(getByRole('checkbox', { name: new RegExp(k.klasse) }));
		}
		expect(senden(getByRole).disabled).toBe(true);
	});

	it('setzt Auswahl und Override-Adresse beim Wiederöffnen zurück', async () => {
		const { getByRole, getByLabelText, rerender } = render(KlassenVersandDialog, {
			...texte,
			open: true,
			klassen,
			onclose: () => {},
			onconfirm: () => {}
		});

		await fireEvent.click(getByRole('checkbox', { name: /05A/ }));
		await fireEvent.input(getByLabelText(/Alternative Empfänger/), {
			target: { value: 'vertretung@schule.de' }
		});
		expect(beschriftung(senden(getByRole))).toBe('2 Klassen anmahnen');

		await rerender({ open: false });
		await rerender({ open: true });

		expect(beschriftung(senden(getByRole))).toBe('3 Klassen anmahnen');
		expect(/** @type {HTMLInputElement} */ (getByLabelText(/Alternative Empfänger/)).value).toBe(
			''
		);
	});
});
