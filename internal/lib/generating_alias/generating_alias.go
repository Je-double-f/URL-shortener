package generatingalias

const char = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func NewGeneratedAliasOneSize(size *int, pointerOne *int) string {
	for *pointerOne < len(char) {
		*pointerOne++
		return string(char[*pointerOne])

	}

	*pointerOne, *size = -1, 2
	return ""
}

func NewGeneratedAliasTwoSize(size *int, pointerOne *int, pointerTwo *int) string {
	for *pointerOne < len(char) {
		*pointerOne++
		for *pointerTwo < len(char) {
			*pointerTwo++
			return string(char[*pointerOne]) + string(char[*pointerTwo])

		}

		*pointerTwo = -1
	}

	*pointerOne, *size = -1, 3
	return ""
}

func NewGeneratedAliasThreeSize(size *int, pointerOne *int, pointerTwo *int, pointerThree *int) string {
	for *pointerOne < len(char) {
		*pointerOne++
		for *pointerTwo < len(char) {
			*pointerTwo++
			for *pointerThree < len(char) {
				*pointerThree++
				return string(char[*pointerOne]) + string(char[*pointerTwo]) + string(char[*pointerThree])

			}
			*pointerThree = -1

		}
		*pointerTwo = -1

	}

	*pointerOne, *size = -1, 4
	return ""
}

func NewGeneratedAliasFourSize(pointerOne *int, pointerTwo *int, pointerThree *int, pointerFour *int) string {
	for *pointerOne < len(char) {
		*pointerOne++
		for *pointerTwo < len(char) {
			*pointerTwo++
			for *pointerThree < len(char) {
				*pointerThree++
				for *pointerFour < len(char) {
					*pointerFour++
					return string(char[*pointerOne]) + string(char[*pointerTwo]) + string(char[*pointerThree]) + string(char[*pointerFour])

				}
				*pointerFour = -1

			}
			*pointerThree = -1

		}
		*pointerTwo = -1

	}

	return ""
}
