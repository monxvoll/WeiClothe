package clothe

// Multipart POST /clothes (Register): field names and limits.
const (
	RegisterMultipartFieldImage       = "image"
	RegisterMultipartFieldGarmentType = "garment_type"
	// Optional: name, source, status (plain form fields).
	MaxRegisterImageBytes = 10 * 1024 * 1024
)
