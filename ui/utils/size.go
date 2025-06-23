package utils

type SizeType uint

const (
	Ratio SizeType = iota
	Fixed
	Modifier
)

type SizeI interface {
	Type() SizeType
}

// ------------ SizeRatio
// Will hold the % with which the size is to be multiplied
// Value between: 0 and 1
type SizeRatio float32

func (sr SizeRatio) Type() SizeType {
	return Ratio
}

// ------------ SizeFixed
// Will hold the absolute value of the size
type SizeFixed int

func (sf SizeFixed) Type() SizeType {
	return Fixed
}

// ------------ SizeModifier
// Will specify how much the size is to be adjusted by.
// e.g. if 4 provides and W is the width => new width = W + 4.
// Values can be set as negative to denote reduction of size.
type SizeModifier int

func (sm SizeModifier) Type() SizeType {
	return Modifier
}
