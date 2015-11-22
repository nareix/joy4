
package atom

type Fixed uint32
type TimeStamp uint32

func IntToFixed(val int) Fixed {
	return Fixed(val<<16)
}

func FixedToInt(val Fixed) int {
	return int(val>>16)
}

