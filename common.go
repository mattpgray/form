package form

import "strconv"

// Some common field parsers

// PositiveInteger parses a form value as a positive integer
type PositiveInteger int

var _ FieldParser = (*PositiveInteger)(nil)

// ParseField implements FieldParser
func (i *PositiveInteger) ParseField(key string, vals []string) error {
	if len(vals) > 0 {
		val, err := strconv.Atoi(vals[0])
		if err != nil {
			return err
		}

		if val < 0 {
			return &UnexpectedValueError{Value: vals[0]}
		}

		*i = PositiveInteger(val)
	}

	return nil
}
