package lacity

import (
	"fmt"
	"strings"
	"time"
	wheretopark "wheretopark/go"

	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
)

type MeterType = string

const (
	MeterTypeSingleSpace MeterType = "Single-Space"
	MeterTypeMultiSpace  MeterType = "Multi-Space"
)

type RateType = string

const (
	RateTypeFlat      RateType = "FLAT"
	RateTypeTimeOfDay RateType = "TOD"
	RateTypeJump      RateType = "JUMP"
)

type Coordinate struct {
	Latitude  float64 `json:"latitude,string"`
	Longitude float64 `json:"longitude,string"`
}

type SpaceMetadata struct {
	SpaceID          string     `json:"spaceid"`
	BlockFace        string     `json:"blockface"`
	MeterType        MeterType  `json:"metertype"`
	RateType         RateType   `json:"ratetype"`
	RateRange        string     `json:"raterange"`
	MeteredTimeLimit string     `json:"timelimit"`
	Coordinate       Coordinate `json:"latlng"`
}

type OccupancyState = string

const (
	OccupancyStateVacant   OccupancyState = "VACANT"
	OccupancyStateOccupied OccupancyState = "OCCUPIED"
)

type SpaceState struct {
	SpaceID        string         `json:"spaceid"`
	EventTime      string         `json:"eventtime"` // ISO8601
	OccupancyState OccupancyState `json:"occupancystate"`
}

func (m *SpaceMetadata) ParseTimeLimit() (string, error) {
	// The values of MeteredTimeLimit can be:
	// 1HR
	// 1HR30MIN
	// 30MIN
	former, latter, _ := strings.Cut(m.MeteredTimeLimit, "-")
	duration := "PT"
	if strings.HasSuffix(former, "HR") {
		var hours uint
		_, err := fmt.Sscanf(former, "%dHR", &hours)
		if err != nil {
			return "", fmt.Errorf("parse hours(%s) fail: %w", former, err)
		}
		duration += fmt.Sprintf("%dH", hours)
	} else if strings.HasSuffix(former, "MIN") {
		var minutes uint
		_, err := fmt.Sscanf(former, "%dMIN", &minutes)
		if err != nil {
			return "", fmt.Errorf("parse minutes(%s) fail: %w", former, err)
		}
		duration += fmt.Sprintf("%dM", minutes)
	} else {
		return "", fmt.Errorf("unknown time limit format: %s", m.MeteredTimeLimit)
	}

	if strings.HasSuffix(latter, "MIN") {
		var minutes uint
		_, err := fmt.Sscanf(latter, "%dMIN", &minutes)
		if err != nil {
			return "", fmt.Errorf("parse latter minutes(%s) fail: %w", latter, err)
		}
		duration += fmt.Sprintf("%dM", minutes)
	}

	return duration, nil
}

func (m *SpaceMetadata) ParseRateFlat() (decimal.Decimal, error) {
	var strRate string
	_, err := fmt.Sscanf(m.RateRange, "$%s", &strRate)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse rate(%s) fail: %w", strRate, err)
	}

	rate, err := decimal.NewFromString(strRate)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse rate(%s) as decimal fail: %w", strRate, err)
	}
	return rate, nil
}

func (m *SpaceMetadata) ParseRateJump() (decimal.Decimal, decimal.Decimal, string, string, error) {
	former, latter, _ := strings.Cut(m.RateRange, " - ")
	var strMinRate, strMaxRate float32
	var maxRateHours uint
	_, err := fmt.Sscanf(former, "$%f/H", &strMinRate)
	if err != nil {
		return decimal.Zero, decimal.Zero, "", "", fmt.Errorf("parse min rate(%s) fail: %w", former, err)
	}

	_, err = fmt.Sscanf(latter, "$%f/%dH", &strMaxRate, &maxRateHours)
	if err != nil {
		return decimal.Zero, decimal.Zero, "", "", fmt.Errorf("parse max rate(%s) fail: %w", former, err)
	}

	minRate := decimal.NewFromFloat32(strMinRate)
	maxRate := decimal.NewFromFloat32(strMaxRate)
	return minRate, maxRate, "PT1H", fmt.Sprintf("PT%dH", maxRateHours), nil
}

func (m *SpaceMetadata) ParseRateTimeOfDay() (decimal.Decimal, decimal.Decimal, error) {
	parts := strings.Split(m.RateRange, " - ")
	var strMinRate, strMaxRate string
	_, err := fmt.Sscanf(parts[0], "$%s", &strMinRate)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("parse min rate(%s) fail: %w", strMinRate, err)
	}

	_, err = fmt.Sscanf(parts[1], "$%s", &strMaxRate)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("parse max rate(%s) fail: %w", strMaxRate, err)
	}
	minRate, err := decimal.NewFromString(strMinRate)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("parse min rate(%s) as decimal fail: %w", strMinRate, err)
	}
	maxRate, err := decimal.NewFromString(strMaxRate)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("parse max rate(%s) as decimal fail: %w", strMaxRate, err)
	}
	return minRate, maxRate, nil
}

func (m *SpaceMetadata) Rules() ([]wheretopark.Rule, error) {
	timeLimit, err := m.ParseTimeLimit()
	if err != nil {
		return nil, fmt.Errorf("failed to parse time limit: %w", err)
	}
	switch m.RateType {
	case RateTypeFlat:
		flatRate, err := m.ParseRateFlat()
		if err != nil {
			return nil, fmt.Errorf("failed to parse flat rate: %w", err)
		}
		return []wheretopark.Rule{
			{
				Hours: "24/7",
				Pricing: []wheretopark.PricingRule{
					{
						Duration: "PT1H",
						Price:    flatRate,
						Limit:    timeLimit,
					},
				},
			},
		}, nil
	case RateTypeJump:
		minRate, maxRate, minRateDuration, maxRateDuration, err := m.ParseRateJump()
		if err != nil {
			return nil, fmt.Errorf("failed to parse jump rate: %w", err)
		}
		return []wheretopark.Rule{
			{
				Hours: "24/7",
				Pricing: []wheretopark.PricingRule{
					{
						Duration:  minRateDuration,
						Price:     minRate,
						Limit:     timeLimit,
						Repeating: true,
					},
					{
						Duration:  maxRateDuration,
						Price:     maxRate,
						Limit:     timeLimit,
						Repeating: true,
					},
				},
			},
		}, nil
	case RateTypeTimeOfDay:
		minRate, maxRate, err := m.ParseRateTimeOfDay()
		if err != nil {
			return nil, fmt.Errorf("failed to parse time of day rate: %w", err)
		}
		return []wheretopark.Rule{
			{
				Hours: "24/7",
				Pricing: []wheretopark.PricingRule{
					{
						Duration:  "PT1H",
						Price:     minRate,
						Limit:     timeLimit,
						Repeating: true,
					},
					{
						Duration:  "PT1H",
						Price:     maxRate,
						Limit:     timeLimit,
						Repeating: true,
					},
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown rate type: %s", m.RateType)
	}
}

const (
	METADATA_URL = "https://data.lacity.org/resource/s49e-q6j2.json"
	STATE_URL    = "https://data.lacity.org/resource/e7h6-4a3e.json"
)

var client = resty.New()

func init() {
	client.GetClient().Timeout = 10 * time.Second
}

func GetSpaceMetadatas() ([]SpaceMetadata, error) {
	resp, err := client.R().SetResult([]SpaceMetadata{}).Get(METADATA_URL)
	if err != nil {
		return nil, err
	}
	response := resp.Result().(*[]SpaceMetadata)
	return *response, nil
}

func GetSpaceStates() ([]SpaceState, error) {
	resp, err := client.R().SetResult([]SpaceState{}).Get(STATE_URL)
	if err != nil {
		return nil, err
	}
	response := resp.Result().(*[]SpaceState)
	return *response, nil
}
