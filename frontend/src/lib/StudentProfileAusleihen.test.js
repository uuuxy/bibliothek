import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('./apiFetch.js', () => ({
	apiFetch: vi.fn(async () => ({ ok: true, json: async () => ({}) })),
	apiClient: { post: vi.fn(async () => ({ ok: true, json: async () => ({}) })) }
}));

import StudentProfileAusleihen from './StudentProfileAusleihen.svelte';

// Rauchtest des Reiters: Ein nicht durchgereichtes Prop ließe still eine ganze Karte leer.
const daten = {
	buecher: [{ id: 'b1', titel: 'Der Hobbit', barcode_id: 'B-1', rueckgabe_frist: '2026-10-01' }],
	vormerkungen: [{ id: 'v1', titel_name: 'Momo', status: 'wartend', erstellt_am: '2026-09-01' }],
	onChanged: () => {}
};

describe('StudentProfileAusleihen', () => {
	it('zeigt Ausleihen und Vormerkungen', () => {
		const screen = render(StudentProfileAusleihen, daten);

		expect(screen.getByText(/Der Hobbit/)).toBeTruthy();
		expect(screen.getByText(/Momo/)).toBeTruthy();
	});

	// Eine leere Karte sähe aus wie „nichts vorgemerkt": Der gescheiterte Abruf steht darüber.
	it('nennt die Listen, deren Abruf gescheitert ist', () => {
		const screen = render(StudentProfileAusleihen, { ...daten, fehlendeListen: ['Vormerkungen'] });
		const hinweis = screen.getByRole('alert');
		expect(hinweis.textContent).toMatch(/Nicht geladen: Vormerkungen/);
	});

	it('schweigt, wenn alles geladen ist', () => {
		const screen = render(StudentProfileAusleihen, { ...daten, fehlendeListen: [] });
		expect(screen.queryByRole('alert')).toBeNull();
	});
});
