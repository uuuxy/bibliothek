import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

// Invariante gegen die "weiße Seite"-Bugklasse: JEDER Wert, der jemals auf
// uiStore.activeTab gesetzt werden kann (Literal-Zuweisung, Sidebar-Menü-ID,
// tabToPath-Deeplink), MUSS vom Router auch gerendert werden. Der White-Screen
// beim Etikettendruck war genau ein Bruch dieser Invariante (App.svelte setzte
// 'labels', der Router kannte nur 'druck-center') — und blieb wochenlang
// unbemerkt, weil kein Gate darauf lief. Dieser Test ist dieses Gate: frei,
// deterministisch, Millisekunden. Kein LLM-Scan nötig.

const libDir = dirname(fileURLToPath(import.meta.url));
const srcRoot = join(libDir, '..');

/** @param {string} p */
function collectSourceFiles(p) {
	/** @type {string[]} */
	const out = [];
	for (const entry of readdirSync(p)) {
		if (entry === 'node_modules') continue;
		const full = join(p, entry);
		if (statSync(full).isDirectory()) out.push(...collectSourceFiles(full));
		else if (/\.(svelte|js)$/.test(entry) && !entry.endsWith('.test.js')) out.push(full);
	}
	return out;
}

/**
 * Reine Prüf-Logik — bewusst als Funktion, damit ein zweiter Test mit
 * synthetischem "kaputtem" Input beweist, dass der Checker einen unbehandelten
 * Tab wirklich meldet (ein grüner Test, der nichts fängt, wäre wertlos).
 * @param {{ routerSrc: string, menuSrc: string, assignmentSrcs: string[] }} sources
 */
function findUnrenderableTabs({ routerSrc, menuSrc, assignmentSrcs }) {
	const rendered = new Set(
		[...routerSrc.matchAll(/uiStore\.activeTab === '([^']+)'/g)].map((m) => m[1])
	);

	const tabToPathBlock = routerSrc.match(/const tabToPath = \{([\s\S]*?)\};/)?.[1] ?? '';
	const tabKeys = [...tabToPathBlock.matchAll(/(?:'([\w-]+)'|(\w[\w-]*))\s*:/g)].map(
		(m) => m[1] || m[2]
	);

	const menuIds = [...menuSrc.matchAll(/id: '([^']+)'/g)].map((m) => m[1]);

	const literals = assignmentSrcs.flatMap((src) =>
		[...src.matchAll(/uiStore\.activeTab\s*=\s*'([^']+)'/g)].map((m) => m[1])
	);

	const targets = new Set([...tabKeys, ...menuIds, ...literals]);
	const unrenderable = [...targets].filter((t) => !rendered.has(t));
	return { rendered, targets, unrenderable };
}

describe('Routing-Konsistenz (activeTab ↔ Router)', () => {
	it('jeder erreichbare activeTab-Wert wird vom Router gerendert', () => {
		const routerSrc = readFileSync(join(libDir, 'Router.svelte'), 'utf8');
		const menuSrc = readFileSync(join(libDir, 'menu.js'), 'utf8');
		const assignmentSrcs = collectSourceFiles(srcRoot).map((f) => readFileSync(f, 'utf8'));

		const { rendered, targets, unrenderable } = findUnrenderableTabs({
			routerSrc,
			menuSrc,
			assignmentSrcs
		});

		// Nicht-leer-Garantie: schützt davor, dass ein kaputtes Regex den Test
		// leer und damit fälschlich grün werden lässt.
		expect(rendered.size).toBeGreaterThan(10);
		expect(targets.size).toBeGreaterThan(10);

		expect(unrenderable).toEqual([]);
	});

	it('Checker meldet einen unbehandelten Tab (Negativ-Beweis)', () => {
		const { unrenderable } = findUnrenderableTabs({
			routerSrc: "uiStore.activeTab === 'a'\nuiStore.activeTab === 'b'\nconst tabToPath = { a: '/a' };",
			menuSrc: "{ id: 'b' }",
			assignmentSrcs: ["uiStore.activeTab = 'ghost'"]
		});
		expect(unrenderable).toContain('ghost');
	});
});
