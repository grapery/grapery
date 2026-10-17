package domain

// Publication selection is separate from the complete reversible comic source.
// Only the server's visual audit creates these identities and acceptance facts.
type FragmentPublicationAsset struct {
	PageIndex int    `json:"pageIndex"`
	ImageURL  string `json:"imageUrl"`
}
type FragmentPublicationPage struct {
	PageIndex int                        `json:"pageIndex"`
	ImageURL  string                     `json:"imageUrl"`
	Quality   *FragmentGenerationQuality `json:"quality"`
}
type FragmentPublicationManifest struct {
	Quality *FragmentGenerationQuality `json:"quality"`
	Pages   []FragmentPublicationPage  `json:"pages"`
}
