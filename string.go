package main

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
