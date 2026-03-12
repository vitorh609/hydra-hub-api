package tickets

import "time"

type Ticket struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	CodTicket   int        `json:"codTicket"`
}

type CreateTicketInput struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueDate     *time.Time `json:"dueDate"`
}

type UpdateTicketInput struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Status      *string    `json:"status"`
	DueDate     *time.Time `json:"dueDate"`
}
