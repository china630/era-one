package store

// Backend is the calendar persistence surface (memory or repo adapter).
type Backend interface {
	Put(owner, uid, body string) *Event
	Get(owner, uid string) (*Event, bool)
	List(owner string) []*Event
	FindByUID(uid string) (*Event, bool)
}

// Ensure Store implements Backend.
var _ Backend = (*Store)(nil)
