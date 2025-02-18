package generatingalias

const char = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func NewGeneratedAliasOneSize(size *int, pointerOne *int) string {
	for *pointerOne < len(char) {
		result := string(char[*pointerOne])
		*pointerOne++
		return result
	}

	*pointerOne = 0
	*size = 2
	return "full"
}

func NewGeneratedAliasTwoSize(size *int, pointerOne *int, pointerTwo *int) string {
	for *pointerOne < len(char) {
		for *pointerTwo < len(char) {
			result := string(char[*pointerOne]) + string(char[*pointerTwo])
			*pointerTwo++
			return result
		}
		*pointerTwo = 0
		*pointerOne++
	}

	*pointerOne, *pointerTwo = 0, 0
	*size = 3
	return "full"
}

func NewGeneratedAliasThreeSize(size *int, pointerOne *int, pointerTwo *int, pointerThree *int) string {
	for *pointerOne < len(char) {
		for *pointerTwo < len(char) {
			for *pointerThree < len(char) {
				result := string(char[*pointerOne]) + string(char[*pointerTwo]) + string(char[*pointerThree])
				*pointerThree++
				return result
			}
			*pointerThree = 0
			*pointerTwo++
		}
		*pointerTwo = 0
		*pointerOne++
	}

	*pointerOne, *pointerTwo, *pointerThree = 0, 0, 0
	*size = 4
	return "full"
}

func NewGeneratedAliasFourSize(size *int, pointerOne *int, pointerTwo *int, pointerThree *int, pointerFour *int) string {
	for *pointerOne < len(char) {
		for *pointerTwo < len(char) {
			for *pointerThree < len(char) {
				for *pointerFour < len(char) {
					result := string(char[*pointerOne]) + string(char[*pointerTwo]) + string(char[*pointerThree]) + string(char[*pointerFour])
					*pointerFour++
					return result
				}
				*pointerFour = 0
				*pointerThree++
			}
			*pointerThree = 0
			*pointerTwo++
		}
		*pointerTwo = 0
		*pointerOne++
	}

	*pointerOne, *pointerTwo, *pointerThree, *pointerFour = 0, 0, 0, 0
	*size = 5
	return "full"
}
