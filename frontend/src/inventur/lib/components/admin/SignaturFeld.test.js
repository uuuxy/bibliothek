import { describe, it, expect, vi } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';

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
		const { container } = render(SignaturFeld, { formular: { signatur: '' } });

		await waitFor(() => expect(container.querySelectorAll('datalist option')).toHaveLength(2));
		expect(container.querySelector('input')?.getAttribute('placeholder')).toBe('z. B. De, Ma');
	});
});
