package migrations

type SortUp []string

func (a SortUp) Len() int {
	return len(a)
}

func (a SortUp) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a SortUp) Less(i, j int) bool {
	iRev, err := Revision(a[i])
	if err != nil {
		return false
	}

	jRev, err := Revision(a[j])
	if err != nil {
		return false
	}

	return iRev < jRev
}

type SortDown []string

func (a SortDown) Len() int {
	return len(a)
}

func (a SortDown) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a SortDown) Less(i, j int) bool {
	iRev, err := Revision(a[i])
	if err != nil {
		return false
	}

	jRev, err := Revision(a[j])
	if err != nil {
		return true
	}

	return jRev < iRev
}
