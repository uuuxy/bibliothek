package repository

// sqlCoverOderDNB ist das Cover eines Titels in den Listen der Bestellung (Zulauf, Bestellsuche,
// Bestellbedarf): sein eigener Eintrag, sonst die Cover-Adresse der DNB zu seiner ISBN, sonst
// leer. Ein leerer Eintrag gilt als keiner. Die Adresse nennt die ISBN ohne Bindestriche: Mit
// ihnen trägt sie noch eine Zeile, die Migration 140 oder 157 neben einer Dublette stehen ließ.
//
// titelAlias ist der Alias der Titelzeile in der umgebenden Abfrage.
func sqlCoverOderDNB(titelAlias string) string {
	isbn := titelAlias + ".isbn"
	return `COALESCE(NULLIF(` + titelAlias + `.cover_url, ''), CASE WHEN ` + isbn + ` IS NOT NULL AND ` + isbn +
		` != '' THEN 'https://portal.dnb.de/opac/mvb/cover?isbn=' || replace(` + isbn + `, '-', '') ELSE '' END)`
}
