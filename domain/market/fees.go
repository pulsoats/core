package market

const PPM int64 = 1_000_000
const SpreadPPM int64 = 200

type TakerMakerFees struct {
	TakerFeeRate int64
	MakerFeeRate int64
}
