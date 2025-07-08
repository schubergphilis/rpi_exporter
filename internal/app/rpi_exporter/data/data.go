package data

type Storer interface {
	Run()
}

type Store struct{}

func (s Store) Run() {}
