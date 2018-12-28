package conditions

type Op int

const (
	OP_EQUAL              Op = 0
	OP_NOT_EQUAL             = 1
	OP_GREATER_THAN          = 2
	OP_GREATER_THAN_EQUAL    = 3
	OP_LESSER_THAN           = 4
	OP_LESSER_THAN_EQUAL     = 5
)
