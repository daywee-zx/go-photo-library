package aimanip

type AiManip interface {
	Embed(input []string) ([][]float32, error)
	TagImage(imagePath string) (TagImageResponse, error)
}
