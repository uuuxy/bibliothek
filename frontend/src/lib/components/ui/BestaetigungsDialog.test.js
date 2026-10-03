import { describe, it, expect } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import BestaetigungsDialog from './BestaetigungsDialog.svelte';
import { bestaetigen, fragen } from '../../stores/bestaetigung.svelte.js';

// Bei „Ist es dasselbe Medium?" lösen beide Knöpfe etwas aus: der eine öffnet den vorhandenen
// Titel, der andere legt einen weiteren an. Wer den Dialog nur schließt, hat nicht geantwortet
// — sonst legte ein Druck auf Escape einen Titel an.
const FRAGE = {
	titel: 'Ist es dasselbe Medium?',
	aktion: 'Titel öffnen',
	abbruch: 'Anderes Medium'
};

/** Stellt die Frage und wartet, bis der Dialog steht.
 * @param {(frage: any) => Promise<any>} stelle */
async function oeffne(stelle) {
	const screen = render(BestaetigungsDialog);
	const antwort = stelle(FRAGE);
	await waitFor(() => screen.getByRole('dialog', { name: FRAGE.titel }));
	return { screen, antwort };
}

describe('BestaetigungsDialog: Antwort und Schließen', () => {
	it('fragen: der Knopf der Aktion ist true', async () => {
		const { screen, antwort } = await oeffne(fragen);
		await fireEvent.click(screen.getByRole('button', { name: 'Titel öffnen' }));
		expect(await antwort).toBe(true);
	});

	it('fragen: der zweite Knopf ist false', async () => {
		const { screen, antwort } = await oeffne(fragen);
		await fireEvent.click(screen.getByRole('button', { name: 'Anderes Medium' }));
		expect(await antwort).toBe(false);
	});

	it('fragen: Escape ist keine Antwort', async () => {
		const { antwort } = await oeffne(fragen);
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(await antwort).toBeNull();
	});

	it('fragen: der Klick neben den Dialog ist keine Antwort', async () => {
		const { screen, antwort } = await oeffne(fragen);
		await fireEvent.click(screen.getByRole('presentation'));
		expect(await antwort).toBeNull();
	});

	it('fragen: eine zweite Frage beantwortet die erste nicht', async () => {
		const { antwort } = await oeffne(fragen);
		const zweite = fragen({ titel: 'Zweite Frage' });
		expect(await antwort).toBeNull();
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(await zweite).toBeNull();
	});

	it('bestaetigen: Escape und der zweite Knopf sind beide „nein"', async () => {
		const erste = await oeffne(bestaetigen);
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(await erste.antwort).toBe(false);
		erste.screen.unmount();

		const zweite = await oeffne(bestaetigen);
		await fireEvent.click(zweite.screen.getByRole('button', { name: 'Anderes Medium' }));
		expect(await zweite.antwort).toBe(false);
	});

	it('bestaetigen: der Knopf der Aktion ist „ja"', async () => {
		const { screen, antwort } = await oeffne(bestaetigen);
		await fireEvent.click(screen.getByRole('button', { name: 'Titel öffnen' }));
		expect(await antwort).toBe(true);
	});
});
