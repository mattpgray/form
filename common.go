package form

import "strconv"

// Some common field parsers

type PositiveInteger int

var _ FieldParser = (*PositiveInteger)(nil)

func (i *PositiveInteger) ParseField(vals []string) error {
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
