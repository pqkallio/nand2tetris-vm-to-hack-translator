package main

func Reverse(s string) string {
	rev := ([]byte)(s)

	for i, j := len(s) - 1, 0; i >= 0; i, j = i-1, j+1 {
		rev[j] = s[i]
	}

	return string(rev)
}

func MultiAppend(slices ...[]string) []string {
	totalLen := 0

	for _, s := range slices {
		totalLen += len(s)
	}

	retVal := make([]string, 0, totalLen)

	for _, s := range slices {
		retVal = append(retVal, s...)
	}

	return retVal
}