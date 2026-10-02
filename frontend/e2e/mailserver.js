import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync, rmSync, writeFileSync } from 'node:fs';

// Der Stack, gegen den die Suite läuft, hat einen Mailserver eingetragen — am
// Entwicklungsrechner den der Schule, übernommen aus der .env. Mehrere Flows gehen Wege,
// die Mail verschicken (Anliegen erledigt, Klassensatz bereit, Alarme). Für die Dauer des
// Laufs zeigt die Einstellung deshalb auf eine Adresse, an der nichts zuhört: Jeder Versand
// endet dort mit „connection refused“.
//
// Der Host bleibt nicht leer: Ohne ihn weicht der Server auf die Umgebung aus
// (mailservice.LadeSMTPKonfig), und dort steht derselbe Mailserver.
export const STUMM = Object.freeze({ host: '127.0.0.1', port: '9' });

/** Trägt, was vor dem Lauf eingetragen war; liegt nur, solange die Einstellung stumm steht. */
export const MERKZETTEL_MAIL = '.e2e-mailserver';

/**
 * Werte gehen als psql-Variablen hinein und stehen im SQL als :'name'. So setzt psql die
 * Anführungszeichen, nicht diese Datei — der Hostname stammt aus der Datenbank.
 * @param {string} sql
 * @param {Record<string, string>} [variablen]
 */
function psql(sql, variablen = {}) {
	const container = process.env.E2E_DB_CONTAINER || 'bibliothek-db-local';
	const argumente = ['exec', '-i', container, 'psql', '-U', 'postgres', '-d', 'bibliothek'];
	argumente.push('-tA', '-v', 'ON_ERROR_STOP=1');
	for (const [name, wert] of Object.entries(variablen)) argumente.push('-v', `${name}=${wert}`);
	return execFileSync('docker', argumente, { input: sql }).toString().trim();
}

/** @returns {{ host: string, port: string } | null} null, wenn die Zeile fehlt */
export function liesMailserver() {
	const zeile = psql(
		`SELECT json_build_object('host', smtp_host, 'port', smtp_port)
		   FROM mail_settings_config WHERE id = 1;`
	);
	return zeile ? JSON.parse(zeile) : null;
}

/** @param {{ host: string, port: string } | null} mailserver */
export function istStumm(mailserver) {
	return mailserver?.host === STUMM.host && mailserver?.port === STUMM.port;
}

/** Vor dem Lauf: merkt sich den eingetragenen Mailserver und stellt die Einstellung stumm. */
export function stelleMailserverStumm() {
	const eingetragen = liesMailserver();
	if (!eingetragen) {
		throw new Error(
			'E2E-Setup: mail_settings_config hat keine Zeile. Ohne sie verschickt der Server mit ' +
				'den Angaben aus der Umgebung, und die lassen sich von hier nicht stummstellen.'
		);
	}
	// Schon stumm heißt: Rest eines abgebrochenen Laufs (der Merkzettel liegt noch und gilt
	// weiter) oder von Hand so eingestellt (dann gibt es nichts zurückzustellen).
	if (istStumm(eingetragen)) return;

	// Erst merken, dann umstellen: Bricht der Lauf dazwischen ab, steht die Einstellung noch.
	writeFileSync(MERKZETTEL_MAIL, JSON.stringify(eingetragen));
	psql(
		`UPDATE mail_settings_config SET smtp_host = :'host', smtp_port = :'port' WHERE id = 1;`,
		STUMM
	);
}

/**
 * Nach dem Lauf: stellt den gemerkten Mailserver zurück, solange die Einstellung noch stumm
 * steht. Was jemand während des Laufs eingetragen hat, bleibt.
 */
export function stelleMailserverZurueck() {
	if (!existsSync(MERKZETTEL_MAIL)) return;
	const vorher = JSON.parse(readFileSync(MERKZETTEL_MAIL, 'utf8'));
	psql(
		`UPDATE mail_settings_config SET smtp_host = :'host', smtp_port = :'port'
		  WHERE id = 1 AND smtp_host = :'stumm_host' AND smtp_port = :'stumm_port';`,
		{ host: vorher.host, port: vorher.port, stumm_host: STUMM.host, stumm_port: STUMM.port }
	);
	rmSync(MERKZETTEL_MAIL, { force: true });
}
