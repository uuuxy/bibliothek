// utils/listenGrenze.js
// Die Grenze einer Liste, die der Server kappt. Er nennt sie im Kopf X-Grenze
// (api/tuer/antwort.go, AntworteListe); die Zahl und die Regel, wann sie gilt, stehen nur dort.
// Führte die Oberfläche eine eigene Zahl, behauptete sie nach einer geänderten Grenze eine falsche
// oder schwiege über eine gekappte Liste.

/**
 * Die Grenze, die der Server für diese Antwort nennt; 0, wenn für sie keine gilt.
 *
 * @param {Response} res
 * @returns {number}
 */
export function grenzeAus(res) {
	return Number(res.headers.get('X-Grenze')) || 0;
}

/**
 * Ob eine Liste gekappt ist: Sie hat die Grenze erreicht, die der Server genannt hat.
 *
 * @param {number} anzahl
 * @param {number} grenze
 * @returns {boolean}
 */
export function istGekappt(anzahl, grenze) {
	return grenze > 0 && anzahl >= grenze;
}
