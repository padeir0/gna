package shared

const (
	Error = iota
	CName
	CRoom
	Num
)

type SrAuth struct {
	UserID uint64
}

type CliAuth struct {
	Name string
}

type Message struct {
	Username string
	Data     string
}

type Cmd struct {
	Data string
	T    int
}
