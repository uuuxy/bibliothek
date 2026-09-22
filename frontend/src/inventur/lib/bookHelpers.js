/**
 * @param {string | null | undefined} subject
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
 * @param {string | null | undefined} subject
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
 * @param {string | null | undefined} dateString
 * @returns {string|null}
 */
export function formatDate(dateString) {
	if (!dateString) return null;
	try {
		const date = new Date(dateString);
		if (Number.isNaN(date.getTime())) return null;
		return new Intl.DateTimeFormat('de-DE', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric'
		}).format(date);
	} catch {
		return null;
	}
}
