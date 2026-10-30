package content

import (
	"net/http"

	"haganeworks/internal/project"
)

func getRow(p *project.Project, table string, id int64, missing string) (project.Row, error) {
	rows, err := p.Rows(table)
	if err != nil {
		return nil, err
	}
	if row := findRow(rows, id); row != nil {
		return row, nil
	}
	return nil, statusErr(http.StatusNotFound, missing)
}

func listRows[T any](p *project.Project, table string, less func(a, b project.Row) bool, convert func(project.Row) T) ([]T, error) {
	rows, err := p.Rows(table)
	if err != nil {
		return nil, err
	}
	if less != nil {
		sortRows(rows, less)
	}
	out := make([]T, 0, len(rows))
	for _, row := range rows {
		out = append(out, convert(row))
	}
	return out, nil
}

func removeRow(table, missing string) func(*project.Project, int64) error {
	return func(p *project.Project, id int64) error {
		return p.Write([]string{table}, func(tx *project.Tx) error {
			if tx.Delete(table, func(row project.Row) bool { return project.Int64(row["id"]) == id }) == 0 {
				return statusErr(http.StatusNotFound, missing)
			}
			return nil
		})
	}
}

func writeRow(p *project.Project, table string, change func(tx *project.Tx) (project.Row, error)) (project.Row, error) {
	var out project.Row
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		row, err := change(tx)
		out = row
		return err
	})
	return out, err
}

func updateRow(p *project.Project, table string, id int64, sets project.Row, check func(tx *project.Tx) error) (project.Row, error) {
	if len(sets) == 0 {
		rows, err := p.Rows(table)
		if err != nil {
			return nil, err
		}
		return findRow(rows, id), nil
	}
	return writeRow(p, table, func(tx *project.Tx) (project.Row, error) {
		if check != nil {
			if err := check(tx); err != nil {
				return nil, err
			}
		}
		return update(tx, table, id, sets)
	})
}
