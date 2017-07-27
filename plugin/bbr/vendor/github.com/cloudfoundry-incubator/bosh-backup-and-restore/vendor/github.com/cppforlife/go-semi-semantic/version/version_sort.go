package version

type AscSorting []Version

func (s AscSorting) Len() int           { return len(s) }
func (s AscSorting) Less(i, j int) bool { return s[i].IsLt(s[j]) }
func (s AscSorting) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
