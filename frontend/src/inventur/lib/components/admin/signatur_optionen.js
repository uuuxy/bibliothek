export const lmfFaecher = [
	'M',
	'E',
	'D',
	'Powie',
	'Erd',
	'Bio',
	'Che',
	'Phy',
	'Ges',
	'Spo',
	'Kun',
	'Mus',
	'Rel',
	'Eth',
	'Info',
	'Spa',
	'Fra',
	'Lat'
];

export const bibKategorien = [
	'Krimi',
	'Fantasy',
	'Sachbuch',
	'Jugendbuch',
	'Kinderbuch',
	'Comic',
	'Manga',
	'Sci-Fi',
	'Historisch',
	'Biografie'
];

export function validateSignatur(sig, track) {
	if (!sig) return true; // Optional

	const isLmf = ['Gymnasium', 'Realschule', 'Hauptschule', 'Förderstufe', 'Oberstufe'].includes(
		track
	);
	const isBib = track === 'Bibliothek';

	if (isLmf) {
		if (!sig.startsWith('LMF ')) return false;
		const fach = sig.substring(4);
		return lmfFaecher.includes(fach);
	} else if (isBib) {
		if (sig.startsWith('BIB ')) {
			return bibKategorien.includes(sig.substring(4));
		}
		return bibKategorien.includes(sig);
	}
	return true;
}
