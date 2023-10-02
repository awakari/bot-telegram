package usage

import "github.com/awakari/client-sdk-go/model/usage"

type Order struct {
	Limit OrderLimit `json:"limit"`
	Price OrderPrice `json:"price"`
}

type OrderLimit struct {
	TimeDays uint32        `json:"timeDays"`
	Count    uint32        `json:"count"`
	Subject  usage.Subject `json:"subject"`
}

type OrderPrice struct {
	Unit  string  `json:"unit"`
	Total float64 `json:"total"`
}
