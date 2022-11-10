package dto

type MQReq struct {
	Queue string `json:"queue"`
	Body  string `json:"body"`
}
