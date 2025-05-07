package Models

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type AttendanceRecord struct {
	RFID     string `json:"rfid"`
	InTime   string `json:"in_time"`
	OutTime  string `json:"out_time,omitempty"`
	Duration string `json:"duration,omitempty"`
}

type StudentPresence struct {
	RFID       string `json:"rfid"`
	Name       string `json:"name"`
	Department string `json:"department"`
	InTime     string `json:"in_time"`
}

type AttendanceHistory struct {
	RFID     string `json:"rfid"`
	Name     string `json:"name"`
	InTime   string `json:"in_time"`
	OutTime  string `json:"out_time,omitempty"`
	Duration string `json:"duration,omitempty"`
}
