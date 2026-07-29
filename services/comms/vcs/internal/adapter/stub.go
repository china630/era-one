package adapter

type LiveKitAdapter interface {
	CreateRoom(name string) (string, error)
	IssueToken(roomName, participant string) (string, error)
}

type Stub struct{}

func (Stub) CreateRoom(name string) (string, error) {
	return "lk-room-" + name, nil
}

func (Stub) IssueToken(roomName, participant string) (string, error) {
	return "lk-token-" + roomName + "-" + participant, nil
}
