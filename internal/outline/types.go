package outline

type UserRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Collection struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Permission  string `json:"permission"`
	UpdatedAt   string `json:"updatedAt"`
}

type Document struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	URL              string   `json:"url"`
	CollectionID     string   `json:"collectionId"`
	ParentDocumentID string   `json:"parentDocumentId"`
	Text             string   `json:"text"`
	Revision         int      `json:"revision"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
	PublishedAt      string   `json:"publishedAt"`
	ArchivedAt       string   `json:"archivedAt"`
	CreatedBy        *UserRef `json:"createdBy"`
	UpdatedBy        *UserRef `json:"updatedBy"`
}

type SearchHit struct {
	Context  string   `json:"context"`
	Document Document `json:"document"`
	Ranking  float64  `json:"ranking"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type CreateDocumentInput struct {
	Title            string `json:"title"`
	CollectionID     string `json:"collectionId,omitempty"`
	ParentDocumentID string `json:"parentDocumentId,omitempty"`
	Text             string `json:"text,omitempty"`
	Publish          *bool  `json:"publish,omitempty"`
}

type UpdateDocumentInput struct {
	ID       string `json:"id"`
	Title    string `json:"title,omitempty"`
	Text     string `json:"text,omitempty"`
	EditMode string `json:"editMode,omitempty"`
	Publish  *bool  `json:"publish,omitempty"`
}

type DocumentFilter struct {
	CollectionID     string
	ParentDocumentID string
	Query            string
	Limit            int
	Offset           int
}

type CollectionFilter struct {
	Query  string
	Limit  int
	Offset int
}

type SearchFilter struct {
	Query  string
	Limit  int
	Offset int
}
