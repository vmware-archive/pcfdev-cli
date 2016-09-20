package cert

type ConcreteSystemStore struct {
	FS        FS
	CmdRunner CmdRunner
}
