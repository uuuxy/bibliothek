import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import LadeFehler from './LadeFehler.svelte';

// Ein Knopf, der nichts tut, ist schlimmer als keiner.
//
// Fund (OFFEN.md 5.12): `ActiveStudentList` reichte `() => {}` durch, wenn es keinen
// Rückruf hatte. „Erneut versuchen" stand da und lief ins Leere — wer ihn drückt, wartet
// auf etwas, das nicht kommt, und hält die Liste für kaputt statt den Knopf.
describe('LadeFehler', () => {
	it('zeigt den Knopf nur, wenn es etwas zu wiederholen gibt', () => {
		const ohne = render(LadeFehler, { props: { titel: 'Nicht geladen', text: 'Grund' } });
		expect(ohne.queryByRole('button', { name: /erneut/i })).toBeNull();

		const mit = render(LadeFehler, {
			props: { titel: 'Nicht geladen', text: 'Grund', onerneut: vi.fn() }
		});
		expect(mit.queryByRole('button', { name: /erneut/i })).not.toBeNull();
	});

	it('ruft den Rückruf beim Klick', async () => {
		const erneut = vi.fn();
		const screen = render(LadeFehler, { props: { onerneut: erneut } });
		screen.getByRole('button', { name: /erneut/i }).click();
		expect(erneut).toHaveBeenCalledTimes(1);
	});
});
