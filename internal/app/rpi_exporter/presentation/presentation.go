package presentation

type Presenter interface {
	Run()
}

type CLI struct{}

func (c CLI) Run() {}
