import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import LmfPlanReihenfolge from './LmfPlanReihenfolge.svelte';

// Zwei Klassen dürfen sich eine Stunde teilen — im Plan der Schule stehen „10R1/10R2"
// und „6F1/6F2" so (Peter, 05.09.2026: „das muss alles super flexibel ablaufen").
// Geprüft wird das Umformen der Reihenfolge selbst: zusammenlegen, wieder trennen,
// schieben, eine Klasse herausnehmen. Die Plätze kommen vom Server; hier steht, WAS
// verteilt wird.
const PLAETZE = [
	{ datum: '2027-06-28', stunde: 3 },
	{ datum: '2027-06-28', stunde: 4 },
	{ datum: '2027-06-28', stunde: 5 },
	{ datum: '2027-06-28', stunde: 6 }
];

// Ohne Markierungen: Diese Tests prüfen die Reihenfolge, nicht die Einordnung der Klassen.
const KEIN_MARKER = { ohneSchueler: () => false };

/** @param {any} zeilen */
function zeige(zeilen, onklasseraus = vi.fn()) {
	return render(LmfPlanReihenfolge, {
		zeilen,
		plaetze: PLAETZE,
		marker: KEIN_MARKER,
		bereit: true,
		ausgelassen: [],
		draussen: () => false,
		markiert: null,
		onklasseraus,
		onhinein: vi.fn()
	});
}

/** Die Klassen je Zeile, wie die Tabelle sie zeigt: eine Klasse als Text, mehrere als
 *  Chips (M3: kein Chip allein) — hier beides als „10R1/10R2" gelesen. */
function klassenJeZeile(container) {
	return [...container.querySelectorAll('tbody tr')].map((tr) =>
		(tr.querySelector('td:nth-child(5)')?.textContent ?? '')
			.trim()
			.split(/\s+/)
			.filter(Boolean)
			.join('/')
	);
}

/** Öffnet das Überlaufmenü einer Zeile und wählt den Eintrag. */
async function menue(getByLabelText, getByRole, nummer, eintrag) {
	await fireEvent.click(getByLabelText(`Aktionen Zeile ${nummer}`));
	await fireEvent.click(getByRole('menuitem', { name: eintrag }));
}

describe('LmfPlanReihenfolge', () => {
	const start = [
		{ klassen: ['10R1'], vermerk: '' },
		{ klassen: ['10R2'], vermerk: '' },
		{ klassen: ['10R3'], vermerk: '' }
	];

	it('legt zwei Klassen in eine Stunde zusammen — und trennt sie wieder', async () => {
		const { container, getByLabelText, getByRole } = zeige(start.map((z) => ({ ...z })));
		await menue(getByLabelText, getByRole, 2, 'Mit der Zeile davor zusammenlegen');
		expect(klassenJeZeile(container)).toEqual(['10R1/10R2', '10R3']);
		// Die zusammengelegte Zeile steht in der Stunde der ersten, 10R3 rückt eine hoch.
		const zeilen = container.querySelectorAll('tbody tr');
		expect(zeilen[0].textContent).toContain('3. Std.');
		expect(zeilen[1].textContent).toContain('4. Std.');

		await menue(getByLabelText, getByRole, 1, 'In einzelne Stunden trennen');
		expect(klassenJeZeile(container)).toEqual(['10R1', '10R2', '10R3']);
	});

	it('schiebt eine Zeile nach unten', async () => {
		const { container, getByLabelText } = zeige(start.map((z) => ({ ...z })));
		await fireEvent.click(getByLabelText('Zeile 1 nach unten'));
		expect(klassenJeZeile(container)).toEqual(['10R2', '10R1', '10R3']);
	});

	it('schiebt eine Zeile an den Anfang und ans Ende — die weiten Wege im Menü', async () => {
		// 60 Zeilen, ein Pfeil je Klick: Der Weg von unten nach oben war 50 Klicks
		// (06.09.2026). Wie im Zeilenmenü einer Playlist gibt es die Sprünge direkt.
		const { container, getByLabelText, getByRole } = zeige(start.map((z) => ({ ...z })));
		await menue(getByLabelText, getByRole, 3, 'An den Anfang');
		expect(klassenJeZeile(container)).toEqual(['10R3', '10R1', '10R2']);
		await menue(getByLabelText, getByRole, 1, 'Ans Ende');
		expect(klassenJeZeile(container)).toEqual(['10R1', '10R2', '10R3']);
	});

	it('plant eine Klasse aus „Noch nicht im Plan" über den Chip ein — und beim Ablegen auf eine Zeile davor', async () => {
		const hinein = vi.fn();
		const { container, getByRole } = render(LmfPlanReihenfolge, {
			zeilen: start.map((z) => ({ ...z })),
			plaetze: PLAETZE,
			marker: KEIN_MARKER,
			bereit: true,
			ausgelassen: ['10R4', '12T1'],
			draussen: (k) => k === '12T1',
			markiert: null,
			onklasseraus: vi.fn(),
			onhinein: hinein
		});
		// Offen steht nur, was ohne Regel fehlt; die Oberstufe liegt eingeklappt.
		await fireEvent.click(getByRole('button', { name: '10R4 einplanen' }));
		expect(hinein).toHaveBeenLastCalledWith('10R4');
		// Ein gezogener Chip landet VOR der Zeile, auf der er abgelegt wird.
		const dritte = container.querySelectorAll('tbody tr')[2];
		await fireEvent.drop(dritte, {
			dataTransfer: {
				getData: (/** @type {string} */ t) => (t === 'text/lmf-klasse' ? '10R4' : '')
			}
		});
		expect(hinein).toHaveBeenLastCalledWith('10R4', 2);
	});

	it('nimmt eine Klasse aus einer geteilten Stunde und meldet sie zurück', async () => {
		const zurueck = vi.fn();
		const { container, getByTitle } = zeige(
			[{ klassen: ['10R1', '10R2'], vermerk: 'zusammen' }],
			zurueck
		);
		await fireEvent.click(getByTitle('10R2 aus dem Plan nehmen'));
		expect(zurueck).toHaveBeenCalledWith('10R2');
		expect(klassenJeZeile(container)).toEqual(['10R1']);
	});

	it('legt eine Zeile fest — vorbelegt mit ihrem Vorschau-Platz — und löst sie wieder', async () => {
		// Die Klasse mit dem Ausflug (Peter, 05.09.2026): „festlegen" macht aus den
		// gerechneten Spalten Eingabefelder, und zwar mit dem Platz, den die Zeile gerade
		// hat — sonst spränge sie beim Klick irgendwohin. „lösen" gibt sie dem Fluss zurück.
		const { container, getByLabelText, getByRole, queryByLabelText } = zeige(
			start.map((z) => ({ ...z }))
		);
		await menue(getByLabelText, getByRole, 2, 'Datum und Stunde festlegen');
		const datum = /** @type {HTMLInputElement} */ (getByLabelText('Fester Tag Zeile 2'));
		expect(datum.value).toBe('2027-06-28');
		expect(container.querySelectorAll('tbody tr')[1].textContent).toContain('4. Std.');
		// Die anderen Zeilen bleiben gerechnet.
		expect(queryByLabelText('Fester Tag Zeile 1')).toBeNull();

		await menue(getByLabelText, getByRole, 2, 'Festen Platz lösen');
		expect(queryByLabelText('Fester Tag Zeile 2')).toBeNull();
	});

	it('entfernt die Zeile, wenn ihre letzte Klasse geht und kein Vermerk bleibt', async () => {
		// Eine einzelne Klasse ist Text, kein Chip — herausgenommen wird sie über das Menü.
		const zurueck = vi.fn();
		const { container, getByLabelText, getByRole } = zeige(
			[{ klassen: ['10R1'], vermerk: '' }],
			zurueck
		);
		expect(container.querySelector('td:nth-child(5) button')).toBeNull();
		await menue(getByLabelText, getByRole, 1, 'Klasse aus dem Plan nehmen');
		expect(zurueck).toHaveBeenCalledWith('10R1');
		expect(container.querySelectorAll('tbody tr')).toHaveLength(0);
	});

	it('nennt ohne ersten Tag, was fehlt — und lässt die gerechneten Spalten leer', () => {
		const { getByTestId, container } = render(LmfPlanReihenfolge, {
			zeilen: [{ klassen: ['10R1'], vermerk: '' }],
			plaetze: [],
			marker: KEIN_MARKER,
			bereit: false,
			ausgelassen: [],
			draussen: () => false,
			markiert: null,
			onklasseraus: vi.fn(),
			onhinein: vi.fn()
		});
		expect(getByTestId('lmf-reihenfolge-hinweis').textContent).toContain('Ersten Tag wählen');
		const zellen = [...container.querySelectorAll('tbody tr td')].slice(1, 4);
		expect(zellen.map((td) => td.textContent?.trim())).toEqual(['', '', '']);
	});
});
