import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import OmniboxTeacherCard from './OmniboxTeacherCard.svelte';

// Die Karte an der Theke beschreibt den Ablauf, nicht den Code: Eine Lehrkraft ist geladen,
// die gescannten Bücher gehen auf sie. Bis zum 16.09.2026 stand dort dreimal „Handapparat" —
// ein Wort aus dem Code (ist_handapparat), nicht von der Theke.
describe('OmniboxTeacherCard', () => {
	it('nennt die Lehrkraft und sagt, wohin die Bücher gehen', () => {
		const screen = render(OmniboxTeacherCard, {
			teacher: { id: 'l1', vorname: 'Karl', nachname: 'Lehmann' },
			onDeselect: vi.fn()
		});
		const text = screen.container.textContent ?? '';

		expect(text).toContain('Karl Lehmann');
		expect(text).toContain('Lehrkraft');
		expect(text).not.toContain('Handapparat');
		expect(screen.getByTitle('Lehrkraft abwählen (ESC)')).toBeTruthy();
	});
});
