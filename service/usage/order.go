package usage

type Order struct {
	Limit OrderLimit `json:"limit"`
	Price OrderPrice `json:"price"`
}

type OrderLimit struct {
	TimeDays uint32 `json:"timeDays"`
	MsgRate  uint32 `json:"msgRate"`
	SubCount uint32 `json:"subCount"`
}

type OrderPrice struct {
	MsgRate  float64 `json:"msgRate"`
	SubCount float64 `json:"subCount"`
	Total    float64 `json:"total"`
	Unit     string  `json:"unit"`
}
