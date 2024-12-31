package utils

type Res struct {
	Message string `json:"message"`
	Ok      bool   `json:"ok"`
	Data    any    `json:"data"`
}
