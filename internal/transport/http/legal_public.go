package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	legalres "github.com/grapestree/fgrapery/grapery/internal/resources/legal"
)

type legalDocPayload struct {
	Key         string `json:"key"`
	Lang        string `json:"lang"`
	Format      string `json:"format"`
	Title       string `json:"title,omitempty"`
	LastUpdated string `json:"last_updated,omitempty"`
	Content     string `json:"content"`
}

// GetLegalTermsOfService serves embedded terms (main API envelope, no auth).
func (h *Handler) GetLegalTermsOfService(c *gin.Context) {
	h.serveLegalDoc(c, legalres.KeyTermsOfService, "terms_of_service")
}

// GetLegalPrivacyPolicy serves embedded privacy policy (main API envelope, no auth).
func (h *Handler) GetLegalPrivacyPolicy(c *gin.Context) {
	h.serveLegalDoc(c, legalres.KeyPrivacyPolicy, "privacy_policy")
}

func (h *Handler) serveLegalDoc(c *gin.Context, docKey, errLabel string) {
	lang := legalres.PreferredLangForAPI(c.Query("lang"), c.GetHeader("Accept-Language"))
	content, chosenLang, err := legalres.Get(docKey, lang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    CodeInternalError,
			Message: "failed to load " + errLabel,
			Data:    nil,
		})
		return
	}

	Success(c, legalDocPayload{
		Key:         docKey,
		Lang:        chosenLang,
		Format:      "markdown",
		Title:       legalres.ExtractTitle(content),
		LastUpdated: legalres.ExtractLastUpdated(content),
		Content:     content,
	})
}
