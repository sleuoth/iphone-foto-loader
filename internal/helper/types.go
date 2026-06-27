package helper

type Device struct {
	UUID        string `json:"uuid"`
	ProductName string `json:"productName"`
	DeviceName  string `json:"deviceName"`
	IsTrusted   bool   `json:"isTrusted"`
}

type IdentifyResponse struct {
	Devices []Device `json:"devices"`
}

type File struct {
	Handle        string  `json:"handle"`
	Name          string  `json:"name"`
	Size          int64   `json:"size"`
	Created       string  `json:"created"`
	MimeType      string  `json:"mimeType"`
	LivePhotoPair *string `json:"livePhotoPair"`
}

type ListResponse struct {
	Files []File `json:"files"`
}
