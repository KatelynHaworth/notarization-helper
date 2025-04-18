package api

type TicketRecord struct {
	RecordName string `json:"recordName"`
	ErrorCode  string `json:"serverErrorCode,omitempty"`
	Fields     struct {
		Ticket SignedTicket `json:"signedTicket,omitempty"`
	} `json:"fields,omitempty"`
}

type SignedTicket struct {
	Value []byte `json:"value"`
}
