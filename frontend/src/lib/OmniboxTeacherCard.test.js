import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import OmniboxTeacherCard from './OmniboxTeacherCard.svelte';

// Die Karte an der Theke beschreibt den Ablauf, nicht den Code: Eine Lehrkraft ist geladen,
// die gescannten Bücher gehen auf sie. Bis zum 16.09.2026 stand dort dreimal „Handapparat" —
// ein Wort aus dem Code (ist_handapparat), nicht von der Theke.
describe('OmniboxTeacherCard', () => {
	it('nennt die Lehrkraft und sagt, wohin die Bücher gehen', () => {
		const screen = render(OmniboxTeacherCard, {
			teacher: { id: 'l1', vorname: 'Karl', nachname: 'Lehmann', art: 'lehrkraft' },
			onDeselect: vi.fn()
		});
		const text = screen.container.textContent ?? '';

		expect(text).toContain('Karl Lehmann');
		expect(text).toContain('Lehrkraft');
		expect(text).not.toContain('Handapparat');
		expect(screen.getByTitle('Abwählen (ESC)')).toBeTruthy();
	});

	// Die Art kommt seit Migration 125 am Leser mit. Stünde hier fest „Lehrkraft", hieße eine
	// LiV an der Theke anders, als sie in der Leserdatei steht.
	it('nennt eine LiV auch so', () => {
		const screen = render(OmniboxTeacherCard, {
			teacher: { id: 'l2', vorname: 'Lea', nachname: 'Vogt', art: 'liv' },
			onDeselect: vi.fn()
		});
		const text = screen.container.textContent ?? '';

		expect(text).toContain('Lea Vogt');
		expect(text).toContain('LiV');
	});
});
