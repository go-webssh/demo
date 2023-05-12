package pkg

type Options struct {
	Addr     string `json:"addr"`
	User     string `json:"user"`
	Password string `json:"password"`
	Cols     int    `json:"cols"`
	Rows     int    `json:"rows"`
}
