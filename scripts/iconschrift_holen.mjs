#!/usr/bin/env node
// Holt die Teilmenge von Googles Material Symbols Outlined für die Namen in
// frontend/src/lib/components/ui/iconNamen.js und legt sie zu den Schriften der Oberfläche.
// Aufruf nach jeder Änderung der Liste: node scripts/iconschrift_holen.mjs
//
// Googles Anleitung (developers.google.com/fonts/docs/material_symbols, „Optimize the icon
// font“ und „Self-hosting the font“): nur die Icons der Anwendung über &icon_names, nur die
// Achsen, die sie nutzt, display=block, und die Datei an einem Ort, den man selbst bestimmt.
// Achsen: opsz folgt der Größe (font-optical-sizing), FILL zeigt den gewählten Zustand, GRAD
// von -25 bis 0 gleicht helle Icons auf dunklem Grund aus (M3, Icons, Grade: „the default grade
// for a dark icon on a light background is 0, and -25 for a light icon on a dark background“).
// Das Gewicht bleibt bei 400.
import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { ICON_NAMEN } from '../frontend/src/lib/components/ui/iconNamen.js';

const ziel = join(dirname(fileURLToPath(import.meta.url)), '../frontend/src/styles/schriften');
// Googles Anleitung verlangt „an alphabetically sorted comma-separated list of icon names“;
// iconNamen.js steht schon so (frontend-hygiene-icons prüft es), und namen.txt hält dieselbe
// Reihenfolge.
const namen = [...ICON_NAMEN];
const adresse =
	'https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,FILL,GRAD@20..48,0..1,-25..0' +
	`&icon_names=${namen.join(',')}&display=block`;
// Google liefert woff2 nur einem Browser, der es lesen kann.
const kopf = {
	'User-Agent':
		'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36'
};

const antwort = await fetch(adresse, { headers: kopf });
const css = await antwort.text();
const datei = /src:\s*url\((https:[^)]+)\)\s*format\('woff2'\)/.exec(css)?.[1];
if (!antwort.ok || !datei) {
	throw new Error(`keine woff2-Datei in der Antwort (${antwort.status}):\n${css}`);
}
const schrift = Buffer.from(await (await fetch(datei, { headers: kopf })).arrayBuffer());
mkdirSync(ziel, { recursive: true });
writeFileSync(join(ziel, 'material-symbols-outlined.woff2'), schrift);
writeFileSync(join(ziel, 'material-symbols-outlined.namen.txt'), `${namen.join('\n')}\n`);
console.log(`${namen.length} Icons, ${schrift.length} Byte`);
