import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import StudentProfileStammdaten from './StudentProfileStammdaten.svelte';

// Die Akte eines Kollegen zeigt NICHT die Felder eines Schülers.
//
// Geburtsdatum, LUSD-Kennung, Postanschrift und Elternadresse gehören einem Kollegen
// nicht. Stünden sie als „Keine Angabe" da, behauptete die Akte, sie FEHLTEN — und
// jemand trüge sie irgendwann nach. Eine Elternadresse an einer Lehrkraft ist kein
// Schönheitsfehler: An sie geht die Vormerkungs- und die Mahnpost.
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

		expect(text).toContain('Persönliche Daten');
		expect(text).toContain('Lehrkraft');
		expect(text).toContain('L-0007');
		expect(text, 'die Postanschrift gehört den Eltern eines Schülers').not.toContain(
			'Postanschrift'
		);
		expect(text).not.toContain('Eltern E-Mail');
		expect(text, 'die LUSD-Kennung kommt aus dem Schülerexport').not.toContain('LUSD');
	});

	it('nennt eine LiV auch so', () => {
		const screen = render(StudentProfileStammdaten, {
			profile: akte('liv'),
			onEdit: vi.fn()
		});
		expect(screen.container.textContent ?? '').toContain('LiV');
	});

	it('sagt, wenn ein Kollege noch keine Ausweisnummer hat', () => {
		const screen = render(StudentProfileStammdaten, {
			profile: { ...akte('lehrkraft'), barcode_id: '' },
			onEdit: vi.fn()
		});
		const text = screen.container.textContent ?? '';
		expect(text).toContain('Noch keine');
		expect(text, 'ohne Nummer gibt es keinen Ausweis zu drucken — das gehört gesagt').toContain(
			'Ausweis'
		);
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
	});
});
