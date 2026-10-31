import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { readFileSync } from 'node:fs';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { srcRoot, sammleQuelldateien, relPfad } from '../hygiene-quellen.js';
import { kopiere } from './zwischenablage.js';
import { snackbars } from '../stores/snackbars.svelte.js';
import BuchKarte from '../../inventur/lib/components/BuchKarte.svelte';

const writeText = vi.fn();

beforeEach(() => {
	writeText.mockReset();
	vi.spyOn(snackbars, 'addToast').mockImplementation(() => 0);
	Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
});

afterEach(() => {
	vi.restoreAllMocks();
	Reflect.deleteProperty(navigator, 'clipboard');
});

describe('kopiere', () => {
	it('sagt ja, wenn der Text in der Zwischenablage steht', async () => {
		writeText.mockResolvedValue(undefined);
		expect(await kopiere('978')).toBe(true);
		expect(writeText).toHaveBeenCalledWith('978');
		expect(snackbars.addToast).not.toHaveBeenCalled();
	});

	it('sagt nein und meldet es, wenn der Browser die Zwischenablage verweigert', async () => {
		writeText.mockRejectedValue(new Error('Write permission denied.'));
		expect(await kopiere('978', 'Link bitte von Hand markieren.')).toBe(false);
		expect(snackbars.addToast).toHaveBeenCalledWith(
			'Kopieren nicht möglich. Link bitte von Hand markieren.',
			'error'
		);
	});

	// Über http gibt es navigator.clipboard gar nicht.
	it('sagt nein und meldet es, wenn es keine Zwischenablage gibt', async () => {
		Reflect.deleteProperty(navigator, 'clipboard');
		expect(await kopiere('978')).toBe(false);
		expect(snackbars.addToast).toHaveBeenCalledWith('Kopieren nicht möglich.', 'error');
	});
});

describe('Die Knöpfe „ISBN kopieren“', () => {
	const buch = {
		id: 'b-1',
		isbn: '9783123456789',
		title: 'Natura',
		author: 'Autorin',
		subject: 'Biologie',
		coverUrl: '',
		gesamt: 1,
		verfuegbar: 1
	};

	it('sagt „Kopiert!“ erst, wenn es gelungen ist', async () => {
		writeText.mockResolvedValue(undefined);
		const { getByRole, findByText } = render(BuchKarte, { titel: buch });
		await fireEvent.click(getByRole('button', { name: /ISBN/ }));
		expect(await findByText('Kopiert!')).toBeTruthy();
	});

	it('sagt nicht „Kopiert!“, wenn es scheitert', async () => {
		writeText.mockRejectedValue(new Error('Write permission denied.'));
		const { getByRole, queryByText } = render(BuchKarte, { titel: buch });
		await fireEvent.click(getByRole('button', { name: /ISBN/ }));
		await tick();
		await new Promise((fertig) => setTimeout(fertig, 0));
		await tick();
		expect(queryByText('Kopiert!')).toBeNull();
		expect(snackbars.addToast).toHaveBeenCalledWith('Kopieren nicht möglich.', 'error');
	});
});

// Vier Stellen schrieben selbst in die Zwischenablage; zwei davon sagten „Kopiert!“, ohne das Ergebnis
// abzuwarten, eine schwieg beim Scheitern.
describe('Tür zur Zwischenablage', () => {
	it('nur dieser Helfer fasst sie an', () => {
		const treffer = sammleQuelldateien(srcRoot)
			.filter((datei) => !datei.endsWith('.test.js'))
			.map((datei) => relPfad(datei))
			.filter((rel) => rel !== 'src/lib/utils/zwischenablage.js')
			.filter((rel) =>
				readFileSync(`${srcRoot}/../${rel}`, 'utf8').includes('navigator.clipboard')
			);
		expect(treffer).toEqual([]);
	});
});
