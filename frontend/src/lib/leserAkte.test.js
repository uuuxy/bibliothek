import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import StudentProfileStammdaten from './StudentProfileStammdaten.svelte';

// EINE Akte für jeden Leser — dieselbe Form wie im Formular.
//
// Diese Datei hat am 16.09.2026 zweimal die Seite gewechselt, und beide Male aus demselben
// Grund: Peter sah sich an, was tatsächlich im Browser steht.
//
// Zuerst prüfte sie, dass ein Kollege KEINE Postanschrift sieht. Das war falsch — an ihr
// hängen Mahnung und Bescheid („ich kann dort aber keine adressedaten etc nachtragen").
//
// Dann prüfte sie, dass er weder Geburtsdatum noch LUSD-Kennung noch Eltern-Adresse sieht.
// Auch das ist gefallen, als das Formular jedem alle Felder anbot: Ein Kollege konnte ein
// Geburtsdatum eintragen, das seine Akte danach nicht zeigte. Ein Feld, das man füllen,
// aber nicht lesen kann, ist schlechter als eines, das leer dasteht.
//
// Was diese Datei JETZT zusagt, ist enger und hält besser: dieselben Angaben für jeden, an
// derselben Stelle. Der Unterschied liegt nur noch bei den HANDLUNGEN — Zusammenführen und
// Löschen sind Schülersachen, solange sie gegen die Sicht `schueler` arbeiten.

describe('Akte eines Lesers', () => {
	/** @param {string} art */
	const akte = (art) => ({
		id: 'l1',
		vorname: 'Katrin',
		nachname: 'Wendlandt',
		art,
		barcode_id: 'L-0007',
		klasse: '',
		geburtsdatum: null,
		lusd_id: null,
		strasse: '',
		eltern_email: ''
	});

	it('zeigt einem Kollegen persönliche Daten statt Stammdaten & Adresse', () => {
		const screen = render(StudentProfileStammdaten, {
			profile: akte('lehrkraft'),
			darfBearbeiten: true,
			darfZusammenfuehren: true,
			onEdit: vi.fn()
		});
		const text = screen.container.textContent ?? '';

		expect(text).toContain('Stammdaten & Adresse');
		expect(text).toContain('Lehrkraft');
		expect(text, 'ohne Postanschrift geht kein Bescheid und keine Mahnung raus').toContain(
			'Postanschrift'
		);
		// Der Hinweis, der nur dem Kollegium gilt: Ein Schüler hat kein Konto.
		expect(text).toContain('Benutzer & Rechte');
	});

	it('zeigt einem Kollegen dieselben Angaben wie einem Schüler', () => {
		/** @param {string} art */
		const angaben = (art) => {
			const screen = render(StudentProfileStammdaten, {
				profile: akte(art),
				darfBearbeiten: true,
				onEdit: vi.fn()
			});
			return [...screen.container.querySelectorAll('p.text-xs')].map((e) => e.textContent?.trim());
		};
		expect(angaben('lehrkraft')).toEqual(angaben('schueler'));
		expect(angaben('liv')).toEqual(angaben('schueler'));
	});

	// Der Fund, mit dem alles anfing: „hier steht nirgends ob jemand ein Schüler, LiV,
	// oder lehrer ist" (Peter, 16.09.2026). Die Art stand nur in der Kollegen-Hälfte der
	// Akte — beim Schüler nirgends. Sie ist die erste Frage an einen Leser und gehört
	// deshalb in JEDE Akte, nicht nur in die der anderen.
	it.each([
		['schueler', 'Schüler'],
		['lehrkraft', 'Lehrkraft'],
		['liv', 'LiV']
	])('nennt bei art=%s die Art „%s" in der Akte', (art, wort) => {
		const screen = render(StudentProfileStammdaten, {
			profile: { ...akte(art), klasse: art === 'schueler' ? '07A' : '' },
			onEdit: vi.fn()
		});
		const text = screen.container.textContent ?? '';
		expect(text).toContain('Art');
		expect(text).toContain(wort);
	});

	// Ein Kollege hatte keinen Bearbeiten-Knopf — deshalb liess sich an seiner Akte
	// nichts nachtragen, egal welches Recht jemand hatte. Der Knopf hängt am Recht,
	// nicht an der Art.
	it('bietet auch einem Kollegen das Bearbeiten an', async () => {
		const onEdit = vi.fn();
		const screen = render(StudentProfileStammdaten, {
			profile: akte('lehrkraft'),
			darfBearbeiten: true,
			onEdit
		});
		const knopf = screen.getByRole('button', { name: /Bearbeiten/ });
		knopf.click();
		expect(onEdit).toHaveBeenCalled();
	});

	it('zeigt den Bearbeiten-Knopf ohne das Recht nicht', () => {
		const screen = render(StudentProfileStammdaten, {
			profile: akte('lehrkraft'),
			darfBearbeiten: false,
			onEdit: vi.fn()
		});
		expect(screen.queryByRole('button', { name: /Bearbeiten/ })).toBeNull();
	});

	it('nennt eine LiV auch so', () => {
		const screen = render(StudentProfileStammdaten, {
			profile: akte('liv'),
			onEdit: vi.fn()
		});
		expect(screen.container.textContent ?? '').toContain('LiV');
	});

	// Zusammenführen: seit dem 16.09.2026 auch beim Kollegium. Vorher blieb der Knopf weg,
	// weil der Schreibweg gegen die Sicht `schueler` lief und bei einem Kollegen null
	// Zeilen traf — ein Knopf, der nichts tut, ist schlimmer als keiner. Jetzt tut er
	// etwas, und er wird gebraucht: Ein doppelt stehender Kollege war sonst von niemandem
	// zu reparieren, auch nicht vom Administrator.
	it('bietet das Zusammenführen bei jedem an, der das Recht hat', () => {
		for (const art of ['schueler', 'lehrkraft', 'liv']) {
			const screen = render(StudentProfileStammdaten, {
				profile: akte(art),
				darfZusammenfuehren: true,
				onEdit: vi.fn()
			});
			expect(screen.container.textContent ?? '', `art=${art}`).toContain('Doppelter Datensatz');
		}
	});

	// Der Grund für ein Doppel ist bei beiden ein anderer, und der Text sagt ihn: beim
	// Schüler der LUSD-Export, beim Kollegen die Selbstanmeldung. Wer den falschen Satz
	// liest, sucht den Fehler an der falschen Stelle.
	it('nennt dem Kollegen den richtigen Grund für ein Doppel', () => {
		const kollege = render(StudentProfileStammdaten, {
			profile: akte('lehrkraft'),
			darfZusammenfuehren: true,
			onEdit: vi.fn()
		}).container.textContent;
		expect(kollege).toContain('Mein Portal');
		// Geprüft wird der SATZ, nicht die Seite: „LUSD ID" steht seit dem 16.09.2026 als
		// Feld in jeder Akte (eine Form für jeden, beim Kollegen leer). Gegen das ganze
		// container.textContent zu prüfen, hiesse dieses Feld mitzumessen — der Test wäre
		// rot, ohne dass an der Erklärung etwas falsch ist.
		expect(kollege, 'die Namensänderung in der LUSD ist eine Schülersache').not.toContain(
			'Namensänderung'
		);

		const schueler = render(StudentProfileStammdaten, {
			profile: akte('schueler'),
			darfZusammenfuehren: true,
			onEdit: vi.fn()
		}).container.textContent;
		expect(schueler).toContain('Namensänderung');
		expect(schueler).not.toContain('Mein Portal');
	});

	it('lässt die Akte eines Schülers, wie sie war', () => {
		const screen = render(StudentProfileStammdaten, {
			profile: { ...akte('schueler'), klasse: '07A', strasse: 'Hauptstraße' },
			onEdit: vi.fn()
		});
		const text = screen.container.textContent ?? '';

		expect(text).toContain('Stammdaten & Adresse');
		expect(text).toContain('Postanschrift');
		expect(text).toContain('Eltern E-Mail');
		expect(text).toContain('LUSD');
		expect(text, 'ein Schüler hat kein Konto — der Hinweis gälte ihm nicht').not.toContain(
			'Benutzer & Rechte'
		);
	});
});
