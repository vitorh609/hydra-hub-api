package notes

import "time"

// Note representa o formato que a API expõe via JSON.
// Observação: a coluna no banco continua `body_text`, mas o JSON exposto usa `content`.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	BodyText  string    `json:"content"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateNoteInput é o payload esperado no POST /notes.
// Campos obrigatórios são não-ponteiro para facilitar validação.
type CreateNoteInput struct {
	Title    string `json:"title"`
	BodyText string `json:"content"`
	Color    string `json:"color"`
}

// UpdateNoteInput é o payload esperado no PUT /notes/{id}.
// Campos são ponteiros para permitir update parcial com COALESCE no SQL.
type UpdateNoteInput struct {
	Title    *string `json:"title"`
	BodyText *string `json:"content"`
	Color    *string `json:"color"`
}
