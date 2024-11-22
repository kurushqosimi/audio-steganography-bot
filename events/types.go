package events

type Fetcher interface {
	Fetch(limit int) ([]Event, error)
}

type Processor interface {
	Process(e Event) error
}

type Type int

const (
	Unknown Type = iota
	Message
	Audio
	Image
)

type Event struct {
	Type    Type
	Text    string
	AudioID string
	Meta    interface{}
}
