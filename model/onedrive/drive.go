package onedrive

type Drive struct {
	Id        string `json:"id"`
	DriveType string `json:"driveType"`
	Quota     struct {
		Total     int64 `json:"total"`
		Used      int64 `json:"used"`
		Remaining int64 `json:"remaining"`
	} `json:"quota"`
	WebUrl string `json:"webUrl"`
}
