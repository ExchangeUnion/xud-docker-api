package bitcoind

type Fork struct {
	Type string
	Active bool
	Height int32
}

type BlockchainInfo struct {
	Chain string
	Blocks int32
	Headers int32
	BestBlockHash string
	Difficulty int64
	MedianTime int32
	VerificationProgress float32
	InitialBlockDownload bool
	ChainWork string
	SizeOnDisk int32
	Pruned bool
	SoftForks map[string]Fork
	Warnings string
}
