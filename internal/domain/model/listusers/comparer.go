package listusers

// Comparer enum definition
type Comparer int32

const (
	ComparerUnspecified      Comparer = 0
	ComparerGreaterThan      Comparer = 1
	ComparerGreaterThanEqual Comparer = 2
	ComparerLessThan         Comparer = 3
	ComparerLessThanEqual    Comparer = 4
	ComparerEqual            Comparer = 5
)
