package constants

// Calendar entry types
type CalendarEntryType string

const (
	EntryTypeDaily          CalendarEntryType = "daily"
	EntryTypeLongterm       CalendarEntryType = "longterm"
	EntryTypeRecommendation CalendarEntryType = "recommendation"
)

// Slot cap per day
const SlotCap = 8

// Ratio configuration per plan type
type SlotRatio struct {
	Daily    int
	Longterm int
	Rec      int
}

var RatioDevelopment = SlotRatio{Daily: 5, Longterm: 2, Rec: 1}
var RatioLearning = SlotRatio{Daily: 3, Longterm: 2, Rec: 3}

func GetRatioForPlanType(planType PlanType) SlotRatio {
	if planType == TypeLearning {
		return RatioLearning
	}
	return RatioDevelopment
}
