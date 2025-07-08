package application

type Executor interface {
	Run()
}

type Execute struct{}

func (e Execute) Run() {}
