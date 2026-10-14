package lusd

// leseCSV ist die schmale Sicht der Parser-Tests auf LeseDatei: die Zeilen und die Liste
// der LUSD-IDs in der Reihenfolge der Datei, ohne leere.
func leseCSV(content []byte) ([]Zeile, []string, error) {
	datei, err := LeseDatei(content)
	if err != nil {
		return nil, nil, err
	}
	var ids []string
	for _, z := range datei.Zeilen {
		if z.LUSDID != "" {
			ids = append(ids, z.LUSDID)
		}
	}
	return datei.Zeilen, ids, nil
}
