export const subjectColors = {
	Mathe: 'bg-blue-50 border border-blue-200 text-blue-700',
	Deutsch: 'bg-red-50 border border-red-200 text-red-700',
	Englisch: 'bg-indigo-50 border border-indigo-200 text-indigo-700',
	Französisch: 'bg-indigo-50 border border-indigo-200 text-indigo-700',
	Geographie: 'bg-emerald-50 border border-emerald-200 text-emerald-700',
	Geschichte: 'bg-amber-50 border border-amber-200 text-amber-700',
	Biologie: 'bg-green-50 border border-green-200 text-green-700',
	Chemie: 'bg-yellow-50 border border-yellow-200 text-yellow-700',
	Physik: 'bg-emerald-50 border border-emerald-200 text-emerald-700',
	Musik: 'bg-pink-50 border border-pink-200 text-pink-700',
	Arbeitslehre: 'bg-orange-50 border border-orange-200 text-orange-700',
	Politik: 'bg-rose-50 border border-rose-200 text-rose-700',
	Informatik: 'bg-cyan-50 border border-cyan-200 text-cyan-700',
	Latein: 'bg-sky-50 border border-sky-200 text-sky-700',
	Spanisch: 'bg-emerald-50 border border-emerald-200 text-emerald-700',
	'kath. Religion': 'bg-violet-50 border border-violet-200 text-violet-700',
	'ev. Religion': 'bg-violet-50 border border-violet-200 text-violet-700',
	Ethik: 'bg-teal-50 border border-teal-200 text-teal-700'
};

/**
 * @param {string} subject
 * @returns {string}
 */
export function getSubjectColor(subject) {
	if (subject in subjectColors) {
		return subjectColors[/** @type {keyof typeof subjectColors} */ (subject)];
	}
	return 'bg-slate-50 border border-slate-200 text-slate-600';
}

/**
 * @param {number} verfuegbar
 * @returns {string}
 */
export function getStockDotColor(verfuegbar) {
	if (verfuegbar === 0) return 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.4)]';
	if (verfuegbar < 5) return 'bg-amber-500 shadow-[0_0_6px_rgba(245,158,11,0.4)]';
	return 'bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.4)]';
}

/**
 * @param {string} subject
 * @returns {string}
 */
export function getSubjectGradient(subject) {
	const clean = (subject || '').trim().toLowerCase();
	if (clean.includes('math'))
		return 'bg-linear-to-br from-blue-600 via-indigo-600 to-blue-700 border-blue-500/30';
	if (clean.includes('deu'))
		return 'bg-linear-to-br from-red-600 via-rose-600 to-red-700 border-red-500/30';
	if (
		clean.includes('eng') ||
		clean.includes('fra') ||
		clean.includes('spa') ||
		clean.includes('lat') ||
		clean.includes('spr')
	)
		return 'bg-linear-to-br from-violet-600 via-purple-600 to-violet-700 border-purple-500/30';
	if (
		clean.includes('bio') ||
		clean.includes('che') ||
		clean.includes('phy') ||
		clean.includes('nat')
	)
		return 'bg-linear-to-br from-teal-600 via-emerald-600 to-teal-700 border-teal-500/30';
	if (
		clean.includes('ges') ||
		clean.includes('pol') ||
		clean.includes('geo') ||
		clean.includes('erd') ||
		clean.includes('soz')
	)
		return 'bg-linear-to-br from-amber-600 via-orange-600 to-amber-700 border-amber-500/30';
	if (clean.includes('mus') || clean.includes('kun'))
		return 'bg-linear-to-br from-pink-600 via-fuchsia-600 to-pink-700 border-pink-500/30';
	if (clean.includes('inf'))
		return 'bg-linear-to-br from-slate-600 via-slate-700 to-slate-800 border-emerald-500/30';
	return 'bg-linear-to-br from-slate-500 via-slate-600 to-slate-700 border-slate-400/30';
}

/**
 * @param {string} subject
 * @returns {string}
 */
export function getSpineGradient(subject) {
	const clean = (subject || '').trim().toLowerCase();
	if (clean.includes('math')) return 'from-blue-300 to-indigo-400';
	if (clean.includes('deu')) return 'from-red-300 to-rose-400';
	if (
		clean.includes('eng') ||
		clean.includes('fra') ||
		clean.includes('spa') ||
		clean.includes('lat') ||
		clean.includes('spr')
	)
		return 'from-violet-300 to-fuchsia-400';
	if (
		clean.includes('bio') ||
		clean.includes('che') ||
		clean.includes('phy') ||
		clean.includes('nat')
	)
		return 'from-teal-300 to-emerald-400';
	if (
		clean.includes('ges') ||
		clean.includes('pol') ||
		clean.includes('geo') ||
		clean.includes('erd') ||
		clean.includes('soz')
	)
		return 'from-amber-300 to-orange-400';
	if (clean.includes('mus') || clean.includes('kun')) return 'from-pink-300 to-fuchsia-400';
	if (clean.includes('inf')) return 'from-emerald-300 to-teal-400';
	return 'from-slate-400 to-slate-500';
}

/**
 * @param {string} dateString
 * @returns {string|null}
 */
export function formatDate(dateString) {
	if (!dateString) return null;
	try {
		const date = new Date(dateString);
		if (isNaN(date.getTime())) return null;
		return new Intl.DateTimeFormat('de-DE', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric'
		}).format(date);
	} catch {
		return null;
	}
}
