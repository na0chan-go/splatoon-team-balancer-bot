package domain

const (
	RatingDeltaStep = 10
	RatingDeltaMin  = -200
	RatingDeltaMax  = 200
)

func ClampRatingDelta(v int) int {
	switch {
	case v < RatingDeltaMin:
		return RatingDeltaMin
	case v > RatingDeltaMax:
		return RatingDeltaMax
	default:
		return v
	}
}

func NextRatingDelta(current int, won bool) int {
	if won {
		return ClampRatingDelta(current + RatingDeltaStep)
	}
	return ClampRatingDelta(current - RatingDeltaStep)
}
