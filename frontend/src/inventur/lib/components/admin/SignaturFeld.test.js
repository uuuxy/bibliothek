import { describe, it, expect, vi } from 'vitest';
import { render, waitFor, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/utils/signaturen.js', () => ({
	ladeSignaturen: vi.fn(async () => [
		{ signatur: 'Ma', titel: 3 },
		{ signatur: 'De', titel: 9 }
	])
}));

import SignaturFeld from './SignaturFeld.svelte';

// Die Signatur wird getippt. Vorschläge und Beispiel kommen aus dem Bestand, damit derselbe
// Regalname nicht in zweiter Schreibweise entsteht.
describe('SignaturFeld', () => {
	it('schlägt die Signaturen des Bestands vor, das größte Regal zuerst', async () => {
		const screen = render(SignaturFeld, { formular: { signatur: '' } });
		const feld = screen.getByRole('combobox', { name: /Signatur/ });

		await waitFor(() => expect(feld.getAttribute('placeholder')).toBe('z. B. De, Ma'));
		await fireEvent.click(screen.getByRole('button', { name: 'Vorschläge zeigen' }));
		expect(
			screen.getAllByRole('option').map((o) => o.textContent?.replace(/\s+/g, ' ').trim())
		).toEqual(['Ma 3 Titel', 'De 9 Titel']);
	});
});
