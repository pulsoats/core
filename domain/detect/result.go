package detect

// DetResult provides Signal, error and detector ID values
type DetResult struct {
	Sig       Signal
	HasSignal bool
	Err       error
	OptsLabel string

	SigIndex int
}

// ResultStream contains DetResult channel and Wait func() for concurrency usage
type ResultStream struct {
	SigCh <-chan DetResult
	Wait  func() error
}
