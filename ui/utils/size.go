package utils

// SizeType - how a SizeI changes a size.
type SizeType uint

const (
	Ratio    SizeType = iota // a SizeRatio: the size times it
	Fixed                    // a SizeFixed: it, whatever the size
	Modifier                 // a SizeModifier: the size plus it
)

// SizeI - a change to a width or a height (see ModifySize).
type SizeI interface {
	Type() SizeType
}

// SizeRatio - the fraction of a size to take, between 0 and 1.
type SizeRatio float32

// Type - Ratio.
func (sr SizeRatio) Type() SizeType {
	return Ratio
}

// SizeFixed - a size in cells, taken whatever the size it changes.
type SizeFixed int

// Type - Fixed.
func (sf SizeFixed) Type() SizeType {
	return Fixed
}

// SizeModifier - the cells to add to a size: 4 makes a width W into W + 4,
// and a negative one makes it smaller.
type SizeModifier int

// Type - Modifier.
func (sm SizeModifier) Type() SizeType {
	return Modifier
}
