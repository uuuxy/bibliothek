// Eine Antwort, wie sie der Server schickt, für Tests. Der Rumpf ist als JSON und als Text
// lesbar, so oft ein Test ihn abruft; eine echte Response gäbe ihn nur einmal her. Ein leerer
// Rumpf ist wie im Browser kein JSON, etwa bei 502 vom Proxy.

/**
 * @param {number} status
 * @param {unknown} [rumpf] geht als JSON hinaus; ohne ihn bleibt der Rumpf leer
 * @returns {any}
 */
export function serverAntwort(status, rumpf) {
	const text = rumpf === undefined ? '' : JSON.stringify(rumpf);
	return {
		ok: status >= 200 && status < 300,
		status,
		headers: new Headers(),
		json: async () => JSON.parse(text),
		text: async () => text
	};
}
