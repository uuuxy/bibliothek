import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { sammleQuelldateien, srcRoot, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Drei Schreibweisen haben einen Helfer: der Betrag in Euro (formatEuro in utils/format.js), das
// Datum (formatDatum, ebenda) und der Text eines gefangenen Fehlers (fehlertext in
// utils/fehlertext.js). Von Hand geschrieben laufen die Stellen auseinander, etwa im Leerzeichen
// vor dem Zeichen, im Tausenderpunkt oder in der Null vor Tag und Monat. Wer eine der drei
// außerhalb ihres Helfers schreibt, wird hier rot.
//
// Ebenso der Vergleich einer Maske mit ihrem Stand vom Öffnen (nurGeaendertes in
// utils/geaendert.js): Er stand in vier Masken von Hand. Der Titel vergleicht in
// inventur/lib/buch_felder.js mit eigener Regel und anderer Form.
//
// Sieht nicht: einen Betrag ohne Leerzeichen vor dem Zeichen in einer Vorlage (`{wert}€`), ein
// Zeichen aus einer Variablen und einen Fehlertext mit eigenem Ersatzsatz
// (`e instanceof Error ? e.message : 'Export fehlgeschlagen'`); der ist gewollt.
const FORMAT = 'src/lib/utils/format.js';
const BETRAG = /\+\s*['"`](?:\s|\\u00a0)*€|\}(?:\s|\\u00a0|&nbsp;)+€|currency\s*:\s*['"`]EUR/;
const FEHLERTEXT =
	/instanceof\s+Error\s*\?\s*[\w$.]+\.message\s*:\s*String\(|String\(\s*[\w$.]+\s+instanceof\s+Error\s*\?|\.message\s*\|\|\s*String\(/;
// Eine Zahl mit Tausenderpunkt schreibt formatZahl; drei Bildschirme hatten dafür einen eigenen
// Helfer, die übrigen schrieben „1234". Sieht nicht: eine Zahl ganz ohne Formatierung.
const ZAHL_VON_HAND = /\.toLocaleString\(\s*['"`]de-DE['"`]\s*\)/;
// Ohne Angaben schreibt toLocaleDateString „T.M.JJJJ" neben „TT.MM.JJJJ" aus formatDatum, ebenso
// toLocaleString an einem neuen Date. Sieht nicht: toLocaleString an einem Datum aus einer
// Variablen und eine eigene Form mit Angaben (der LMF-Plan schreibt „TT.MM.JJ").
const DATUM_OHNE_ANGABEN =
	/\.toLocaleDateString\(\s*[^,()]*\)|new\s+Date\([^()]*\)\s*\.toLocaleString\(\s*[^,()]*\)/;

// Ein gefangener Wert roh in einem Text (`${e}`, String(e), '…' + e) wird zu „Error: …", ein
// Netzfehler zu „TypeError: Failed to fetch"; seine message direkt in der Meldung ist der
// Fehlertext von Hand. Sieht nicht: den Wert, nachdem er in eine andere Variable gewandert ist,
// und einen Fehler, der als Parameter statt über catch ankommt.
const FANG = /\bcatch\s*\(\s*([\w$]+)\s*\)|\.catch\(\s*\(?\s*([\w$]+)\s*\)?\s*=>/g;
/** @param {string} quelle */
function meldetGefangenesOhneHelfer(quelle) {
	return [...quelle.matchAll(FANG)].some(([, inCatch, inRueckruf]) => {
		const n = (inCatch ?? inRueckruf).replaceAll('$', '\\$');
		const roh = `\\$\\{\\s*${n}\\s*\\}|(?<![\\w$.])String\\(\\s*${n}\\s*\\)|\\+\\s*${n}(?![\\w$.[(])`;
		const message = `(?:addToast|melde)\\(\\s*(?:\\/\\*\\*[^*]*\\*\\/\\s*)?\\(?\\s*${n}\\s*\\)?\\.message\\b`;
		return new RegExp(`${roh}|${message}`).test(quelle);
	});
}
const GEAENDERT = 'src/lib/utils/geaendert.js';
const VERGLEICH_MIT_GELADEN = /!==\s*(?:[\w$]+\.)*geladen\[/;

// Die Meldung einer fehlgeschlagenen Antwort liest leseApiFehler (apiFetch.js), samt den übrigen
// Angaben leseApiFehlerMitDaten. Der Server antwortet mit {"error": "…"}; wer den Rumpf selbst
// las, zeigte die Klammern mit an, packte ihn nach eigener Regel aus oder las res.json() vor der
// Prüfung auf res.ok, und ein leerer Rumpf hieß „Unexpected end of JSON input". Sieht nicht:
// einen Rumpf unter anderem Namen als res, response oder antwort, und einen JSON-Rumpf, der über
// einen Helfer oder ein Objekt weitergereicht wird, bevor jemand error liest.
const API_FETCH = 'src/lib/apiFetch.js';
const RUMPF_ALS_TEXT = /\b(?:res|response|antwort)\s*\.\s*text\(\)/;
const JSON_NAME = /(?:const|let)\s+([\w$]+)\s*=\s*await\s+[\w$.]+\.json\(/g;
const ERROR_IM_AUSDRUCK =
	/\.json\(\)[^;]*?\)\s*\??\.error\b|\.json\(\)\s*\.then\(\s*\(?\s*([\w$]+)\s*\)?\s*=>\s*\1\??\.error\b/;
/** @param {string} quelle */
function liestErrorAusJson(quelle) {
	if (ERROR_IM_AUSDRUCK.test(quelle)) return true;
	return [...quelle.matchAll(JSON_NAME)].some(([, name]) =>
		new RegExp(`(?<![\\w$.])${name.replaceAll('$', '\\$')}\\s*\\??\\.error\\b`).test(quelle)
	);
}

describe('Betrag und Fehlertext nur über ihre Helfer', () => {
	const dateien = sammleQuelldateien(srcRoot);
	/** @param {RegExp} muster */
	const treffer = (muster) =>
		dateien.filter((f) => muster.test(ohneKommentare(readFileSync(f, 'utf8')))).map(relPfad);

	it('durchsucht überhaupt Dateien', () => {
		expect(dateien.length).toBeGreaterThan(300);
	});

	it('keine Quelldatei außer utils/format.js schreibt einen Betrag in Euro von Hand', () => {
		const von_hand = treffer(BETRAG);
		// Sähe der Detektor die erlaubte Stelle nicht, mäße er nichts.
		expect(von_hand, 'der Detektor findet formatEuro nicht').toContain(FORMAT);
		expect(
			von_hand.filter((f) => f !== FORMAT),
			'Betrag von Hand geschrieben — formatEuro aus utils/format.js nehmen'
		).toEqual([]);
	});

	it('keine Quelldatei außer utils/format.js schreibt eine Zahl mit Tausenderpunkt von Hand', () => {
		const von_hand = treffer(ZAHL_VON_HAND);
		expect(von_hand, 'der Detektor findet formatZahl nicht').toContain(FORMAT);
		expect(
			von_hand.filter((f) => f !== FORMAT),
			'Zahl von Hand geschrieben — formatZahl aus utils/format.js nehmen'
		).toEqual([]);
		expect(ZAHL_VON_HAND.test("const num = (v) => (v ?? 0).toLocaleString('de-DE');")).toBe(true);
		expect(ZAHL_VON_HAND.test('>{t.val.toLocaleString("de-DE")}</text')).toBe(true);
		expect(ZAHL_VON_HAND.test("d.toLocaleString('de-DE', { hour: '2-digit' })")).toBe(false);
	});

	it('keine Quelldatei schreibt ein Datum ohne Angaben, also mit einstelligem Tag und Monat', () => {
		expect(
			treffer(DATUM_OHNE_ANGABEN),
			'Datum ohne Angaben geschrieben („2.1.2026") — formatDatum aus utils/format.js nehmen'
		).toEqual([]);
		for (const form of [
			"{new Date(titel.lastCounted).toLocaleDateString('de-DE')}",
			'`Erstellt am: ${jetzt.toLocaleDateString("de-DE")}`',
			'const text = last.toLocaleDateString();',
			'd.toLocaleDateString( undefined )',
			"new Date(iso).toLocaleString('de-DE')"
		]) {
			expect(DATUM_OHNE_ANGABEN.test(form), form).toBe(true);
		}
		for (const form of [
			"d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' })",
			"(wert ?? 0).toLocaleString('de-DE')",
			'formatDatum(titel.lastCounted)'
		]) {
			expect(DATUM_OHNE_ANGABEN.test(form), form).toBe(false);
		}
	});

	it('keine Quelldatei schreibt den Text eines gefangenen Fehlers von Hand', () => {
		expect(
			treffer(FEHLERTEXT),
			'Fehlertext von Hand geschrieben — fehlertext aus utils/fehlertext.js nehmen'
		).toEqual([]);
	});

	it('keine Quelldatei setzt einen gefangenen Wert ohne fehlertext in eine Meldung', () => {
		expect(
			dateien
				.filter((f) => meldetGefangenesOhneHelfer(ohneKommentare(readFileSync(f, 'utf8'))))
				.map(relPfad),
			'Gefangener Wert ohne Helfer in der Meldung („Error: …") — fehlertext aus utils/fehlertext.js nehmen'
		).toEqual([]);
		for (const form of [
			"} catch (e) {\n\tmelde(`Netzwerkfehler beim Versand: ${e}`, 'error');",
			".catch((e) => melde(`${e}`, 'error'));",
			'.catch(e => { fehler = `${ e }`; });',
			'} catch (err) {\n\tladefehler = `Die Liste fehlt (${err}).`;',
			'} catch (e) {\n\tf.error = String(e);',
			"} catch (fehler) {\n\tmelde('Fehler: ' + fehler, 'error');",
			"} catch (err) {\n\tmelde(/** @type {any} */ (err).message, 'error');",
			"} catch (e) {\n\tsnackbars.addToast(e.message, 'error');",
			"} catch (e) {\n\tmelde(e.message + ' (gekürzt)', 'error');"
		]) {
			expect(meldetGefangenesOhneHelfer(form), form).toBe(true);
		}
		for (const form of [
			"} catch (e) {\n\tmelde(fehlertext(e), 'error');",
			"} catch (e) {\n\tmelde(`Netzwerkfehler: ${fehlertext(e)}`, 'error');",
			"} catch (e) {\n\tconsole.error('Laden fehlgeschlagen', e);",
			'} catch (err) {\n\tthrow err instanceof Error ? err : new Error(fehlertext(err));',
			"} catch (e) {\n\tconsole.error('Abbruch: ' + e.stack);",
			"} catch (e) {\n\tconsole.warn(e);\n}\nmelde(json.message ?? 'Gesendet', 'success');"
		]) {
			expect(meldetGefangenesOhneHelfer(form), form).toBe(false);
		}
	});

	it('keine Maske vergleicht von Hand mit ihrem Stand vom Öffnen', () => {
		const von_hand = treffer(VERGLEICH_MIT_GELADEN);
		expect(von_hand, 'der Detektor findet nurGeaendertes nicht').toContain(GEAENDERT);
		expect(
			von_hand.filter((f) => f !== GEAENDERT),
			'Vergleich mit dem geladenen Stand von Hand — nurGeaendertes aus utils/geaendert.js nehmen'
		).toEqual([]);
		for (const form of [
			'.filter(([name, wert]) => wert !== form.geladen[name])',
			'AENDERBAR.filter((name) => form[name] !== form.geladen[name])',
			'.filter((name) => jetzt[name] !== geladen[name])'
		]) {
			expect(VERGLEICH_MIT_GELADEN.test(form), form).toBe(true);
		}
		expect(VERGLEICH_MIT_GELADEN.test('return nurGeaendertes(form.geladen, felder);')).toBe(false);
	});

	it('keine Quelldatei außer apiFetch.js liest den Rumpf einer Antwort als Text', () => {
		const von_hand = treffer(RUMPF_ALS_TEXT);
		expect(von_hand, 'der Detektor findet leseApiFehler nicht').toContain(API_FETCH);
		expect(
			von_hand.filter((f) => f !== API_FETCH),
			'Rumpf als Text gelesen — die Meldung liest leseApiFehler aus apiFetch.js'
		).toEqual([]);
		for (const form of [
			"throw new Error((await res.text()) || 'Fehler beim Laden');",
			'const errText = await response.text();',
			'const leseRumpf = (res) => (res.ok ? res.json() : res.text());',
			'const text = await antwort . text();'
		]) {
			expect(RUMPF_ALS_TEXT.test(form), form).toBe(true);
		}
		expect(RUMPF_ALS_TEXT.test('const inhalt = await file.text();')).toBe(false);
		expect(RUMPF_ALS_TEXT.test('throw new Error(await leseApiFehler(res));')).toBe(false);
	});

	it('keine Quelldatei außer apiFetch.js liest error aus einer JSON-Antwort', () => {
		expect(
			dateien
				.filter((f) => liestErrorAusJson(ohneKommentare(readFileSync(f, 'utf8'))))
				.map(relPfad)
				.filter((f) => f !== API_FETCH),
			'error aus der Antwort gelesen — leseApiFehler oder leseApiFehlerMitDaten aus apiFetch.js'
		).toEqual([]);
		for (const form of [
			"const err = await res.json().catch(() => ({}));\nmelde(err.error || 'Fehler', 'error');",
			"const daten = await res.json();\nif (!res.ok) throw new Error(daten.error || 'Fehler');",
			'const d = await response.json();\nif (d?.error) return String(d.error);',
			"melde((await res.json().catch(() => ({}))).error || 'Fehler beim Löschen', 'error');",
			'const grund = await res\n\t.json()\n\t.then(( d) => d?.error)\n\t.catch(() => null);'
		]) {
			expect(liestErrorAusJson(form), form).toBe(true);
		}
		for (const form of [
			"const daten = await res.json();\nmelde(daten.message, 'success');",
			"const { meldung, daten } = await leseApiFehlerMitDaten(res, 'Fehler');",
			"f.error = await leseApiFehler(res, 'Fehler beim Senden.');",
			"const json = await res.json();\nformular.error = 'Pflichtfeld';"
		]) {
			expect(liestErrorAusJson(form), form).toBe(false);
		}
	});

	it('Selbstprobe: die Detektoren fassen die Formen', () => {
		for (const form of [
			"return n.toLocaleString('de-DE', { minimumFractionDigits: 2 }) + ' €';",
			"const euro = (b) => b.toFixed(2).replace('.', ',') + ' €';",
			'const text = wert.toFixed(2) + "\\u00a0€";',
			'const satz = `Die Forderung über ${betrag} € wurde storniert.`;',
			'<td>{zeile.summe.toFixed(2)} €</td>',
			'<td>{zeile.summe.toFixed(2)}&nbsp;€</td>',
			"wert.toLocaleString('de-DE', { style: 'currency', currency: 'EUR' })"
		]) {
			expect(BETRAG.test(form), form).toBe(true);
		}
		// Das Zeichen allein hinter einem Feld ist kein Betrag.
		expect(BETRAG.test('{#snippet nachlaufend()}€{/snippet}')).toBe(false);
		expect(BETRAG.test('<span class="text-sm">€</span>')).toBe(false);

		for (const form of [
			'error = err instanceof Error ? err.message : String(err);',
			'melde(fehler instanceof Error ? fehler.message : String(fehler));',
			'fehler = String(e instanceof Error ? e.message : e);',
			'fehler = /** @type {any} */ (err).message || String(err);',
			"snackbars.addToast(err.message || String(err), 'error');"
		]) {
			expect(FEHLERTEXT.test(form), form).toBe(true);
		}
		expect(FEHLERTEXT.test("const text = e.message || 'Unbekannter Fehler';")).toBe(false);
		expect(
			FEHLERTEXT.test("return e instanceof Error ? e.message : 'Zuordnen fehlgeschlagen';")
		).toBe(false);
		expect(FEHLERTEXT.test('if (e instanceof Error) return e.message;')).toBe(false);
	});
});
