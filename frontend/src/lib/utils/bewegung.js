// Die Kurven von Material 3 als Funktion: Svelte-Übergänge nehmen `easing` als Funktion, nicht
// als cubic-bezier(). Werte aus M3 Motion, Easing and duration, Tokens & specs (Zeile CSS).

/**
 * @param {number} x1
 * @param {number} y1
 * @param {number} x2
 * @param {number} y2
 * @returns {(t: number) => number}
 */
function kurve(x1, y1, x2, y2) {
	/** @param {number} a @param {number} b @param {number} s */
	const bezier = (a, b, s) => 3 * a * s * (1 - s) ** 2 + 3 * b * s * s * (1 - s) + s ** 3;
	return (t) => {
		if (t <= 0) return 0;
		if (t >= 1) return 1;
		// x(s) steigt, weil x1 und x2 zwischen 0 und 1 liegen; Halbieren findet s also sicher.
		let unten = 0;
		let oben = 1;
		for (let i = 0; i < 30; i++) {
			const mitte = (unten + oben) / 2;
			if (bezier(x1, x2, mitte) < t) unten = mitte;
			else oben = mitte;
		}
		return bezier(y1, y2, (unten + oben) / 2);
	};
}

/** Kleines Element tritt ein, 250 ms („Standard decelerate“). */
export const standardDecelerate = kurve(0, 0, 0, 1);
/** Kleines Element tritt aus, 200 ms („Standard accelerate“). */
export const standardAccelerate = kurve(0.3, 0, 1, 1);
/** Fläche tritt auf den Bildschirm, 400 ms („Emphasized decelerate“). */
export const emphasizedDecelerate = kurve(0.05, 0.7, 0.1, 1);
/** Fläche verlässt den Bildschirm, 200 ms („Emphasized accelerate“). */
export const emphasizedAccelerate = kurve(0.3, 0, 0.8, 0.15);
