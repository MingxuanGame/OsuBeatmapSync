package onedrive

type ErrorResponse struct {
	Error struct {
		Code       string `json:"code"`
		Message    string `json:"message"`
		InnerError struct {
			Code string `json:"code"`
		} `json:"innererror"`
		Details []interface{} `json:"details"`
	} `json:"error"`
}
