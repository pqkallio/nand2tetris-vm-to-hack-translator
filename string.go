package main

func Reverse(s string) string {
	rev := ([]byte)(s)

	for i, j := len(s) - 1, 0; i >= 0; i, j = i-1, j+1 {
		rev[j] = s[i]
	}

	return string(rev)
}
