package truenas

// QueryFilter is a TrueNAS query filter triple: [field, op, value].
type QueryFilter struct {
	Field string
	Op    string
	Value any
}

func filtersToRaw(filters []QueryFilter) []any {
	raw := make([]any, len(filters))
	for i, f := range filters {
		raw[i] = []any{f.Field, f.Op, f.Value}
	}
	return raw
}
