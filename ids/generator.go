package ids

type Generator interface {
	GenerateID() (string, error)
}
