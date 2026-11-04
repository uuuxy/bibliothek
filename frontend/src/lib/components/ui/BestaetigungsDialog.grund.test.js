import { describe, it, expect } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import BestaetigungsDialog from './BestaetigungsDialog.svelte';
import { loeschenMitGrund } from '../../stores/bestaetigung.svelte.js';

// Löschen mit Grund: „Alter Bestand" ist vorgewählt, ein Klick auf „Löschen" genügt. Verliehene
// Exemplare stehen mit Namen darüber und gehen nur mit, wenn ihr eigenes Kästchen angekreuzt ist;
// der Knopf sagt dann, was geschieht.
const VERLIEHEN = {
	hinweis: '2 Exemplare sind verliehen:',
	liste: ['Anna Muster (7b)', 'Ben Beispiel (8a)'],
	nachsatz: 'Sie bleiben bei den Lesern, und der Titel bleibt, bis sie zurück sind.',
	kaestchen: 'Auch diese 2 Exemplare löschen und aus den Leserkonten austragen',
	aktionMit: 'Alle 5 Exemplare löschen'
};

/** @param {any} frage */
async function oeffne(frage) {
	const screen = render(BestaetigungsDialog);
	const antwort = loeschenMitGrund(frage);
	await waitFor(() => screen.getByRole('dialog', { name: frage.titel }));
	return { screen, antwort };
}

describe('BestaetigungsDialog: Löschen mit Grund', () => {
	it('wählt „Alter Bestand" vor und antwortet damit', async () => {
		const { screen, antwort } = await oeffne({ titel: 'Exemplar B-1 löschen?' });
		expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Alter Bestand')).checked).toBe(
			true
		);
		expect(screen.getAllByRole('radio')).toHaveLength(4);
		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));
		expect(await antwort).toEqual({ grund: 'AUSSORTIERT', zusatz: false });
	});

	it('nimmt einen anderen Grund', async () => {
		const { screen, antwort } = await oeffne({ titel: 'Exemplar B-1 löschen?' });
		await fireEvent.click(screen.getByLabelText('Beschädigt'));
		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));
		expect(await antwort).toEqual({ grund: 'BESCHAEDIGUNG', zusatz: false });
	});

	it('Abbrechen löscht nichts, und der Fokus steht darauf', async () => {
		const { screen, antwort } = await oeffne({ titel: 'Exemplar B-1 löschen?' });
		const abbrechen = screen.getByRole('button', { name: 'Abbrechen' });
		await waitFor(() => expect(document.activeElement).toBe(abbrechen));
		await fireEvent.click(abbrechen);
		expect(await antwort).toBeNull();
	});

	it('nennt die Leser; erst das Kästchen nimmt die verliehenen mit, und der Knopf sagt es', async () => {
		const { screen, antwort } = await oeffne({
			titel: '„Mathematik 7“ löschen?',
			aktion: '3 Exemplare löschen',
			zusatz: VERLIEHEN
		});
		expect(screen.getByText('Anna Muster (7b)')).toBeTruthy();
		expect(screen.getByRole('button', { name: '3 Exemplare löschen' })).toBeTruthy();
		await fireEvent.click(screen.getByLabelText(VERLIEHEN.kaestchen));
		await fireEvent.click(screen.getByRole('button', { name: 'Alle 5 Exemplare löschen' }));
		expect(await antwort).toEqual({ grund: 'AUSSORTIERT', zusatz: true });
	});

	it('sperrt „Löschen", solange ohne das Kästchen nichts zu löschen ist', async () => {
		const { screen, antwort } = await oeffne({
			titel: '„Mathematik 7“ löschen?',
			zusatz: { ...VERLIEHEN, aktionMit: 'Alle 2 Exemplare löschen', noetig: true }
		});
		const loeschen = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: 'Löschen' })
		);
		expect(loeschen.disabled).toBe(true);
		await fireEvent.click(screen.getByLabelText(VERLIEHEN.kaestchen));
		const alle = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: 'Alle 2 Exemplare löschen' })
		);
		expect(alle.disabled).toBe(false);
		await fireEvent.click(alle);
		expect(await antwort).toEqual({ grund: 'AUSSORTIERT', zusatz: true });
	});

	it('beginnt jede neue Frage mit Vorauswahl und leerem Kästchen', async () => {
		const erste = await oeffne({ titel: '„Erster“ löschen?', zusatz: VERLIEHEN });
		await fireEvent.click(erste.screen.getByLabelText('Verlust'));
		await fireEvent.click(erste.screen.getByLabelText(VERLIEHEN.kaestchen));
		const zweite = loeschenMitGrund({ titel: '„Zweiter“ löschen?', zusatz: VERLIEHEN });
		await waitFor(() => erste.screen.getByRole('dialog', { name: '„Zweiter“ löschen?' }));
		expect(await erste.antwort, 'die zweite Frage beantwortet die erste mit nein').toBeNull();
		expect(
			/** @type {HTMLInputElement} */ (erste.screen.getByLabelText('Alter Bestand')).checked
		).toBe(true);
		expect(
			/** @type {HTMLInputElement} */ (erste.screen.getByLabelText(VERLIEHEN.kaestchen)).checked
		).toBe(false);
		await fireEvent.click(erste.screen.getByRole('button', { name: 'Abbrechen' }));
		expect(await zweite).toBeNull();
	});
});
