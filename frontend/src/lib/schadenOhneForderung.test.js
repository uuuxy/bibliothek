import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import DamageReportModal from './DamageReportModal.svelte';

// Ein Kollege bekommt keine Forderung (entschieden am 16.09.2026). Entschieden wird das am
// Server (repository/schaden_melden.go); der Dialog darf dann aber auch nicht nach einem
// Betrag fragen, den niemand fordern wird — sonst tippt die Bibliothekskraft eine Zahl ein,
// die spurlos verschwindet, und wartet danach auf einen Bescheid, der nie kommt.
describe('Verlust/Schaden melden', () => {
	const buch = { id: 'e1', ausleihe_id: 'a1', titel: 'Momo', barcode_id: 'B-1' };

	it('fragt bei einem Kollegen nicht nach einem Ersatzbetrag', () => {
		const screen = render(DamageReportModal, {
			book: { ...buch, ohneForderung: true },
			onCancel: vi.fn(),
			onSubmit: vi.fn()
		});
		const text = screen.container.textContent ?? '';

		expect(screen.queryByLabelText(/Ersatzbetrag/)).toBeNull();
		expect(text).toContain('Eine Forderung entsteht nicht');
		expect(text, 'der Bestand wird trotzdem geführt').toContain('ausgesondert');
	});

	it('schickt bei einem Kollegen den Betrag 0', async () => {
		const onSubmit = vi.fn();
		const screen = render(DamageReportModal, {
			book: { ...buch, ohneForderung: true },
			onCancel: vi.fn(),
			onSubmit
		});
		await fireEvent.click(screen.getByText('Melden'));

		expect(onSubmit).toHaveBeenCalledWith('Verloren', 0, 'nicht_zurueckgegeben');
	});

	it('fragt bei einem Schüler weiter nach dem Ersatzbetrag', () => {
		const screen = render(DamageReportModal, {
			book: buch,
			onCancel: vi.fn(),
			onSubmit: vi.fn()
		});

		expect(screen.getByLabelText(/Ersatzbetrag/)).not.toBeNull();
		expect(screen.container.textContent ?? '').toContain('Forderung angelegt');
	});
});
