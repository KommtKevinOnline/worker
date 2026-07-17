package models

type PredictionType int

const (
	Live PredictionType = iota
	Uncertain
	Offday
)

var predictionTypeName = map[PredictionType]string{
	Live:      "live",
	Uncertain: "uncertain",
	Offday:    "offday",
}

func (pt PredictionType) String() string {
	return predictionTypeName[pt]
}
