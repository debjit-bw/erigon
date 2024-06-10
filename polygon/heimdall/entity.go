package heimdall

type Entity interface {
	RawId() uint64
	BlockNumRange() ClosedRange
	SetRawId(uint64)
}
