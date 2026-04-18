package pay

import (
	"net/http"

	"github.com/gin-gonic/gin"

	legalres "github.com/grapestree/fgrapery/grapery/internal/resources/legal"
)

type legalDocData struct {
	Key         string `json:"key"`
	Lang        string `json:"lang"`
	Format      string `json:"format"`
	Title       string `json:"title,omitempty"`
	LastUpdated string `json:"last_updated,omitempty"`
	Content     string `json:"content"`
}

func GetTermsOfService(c *gin.Context) {
	lang := legalres.ParsePreferredLang(c.GetHeader("Accept-Language"), c.Query("lang"))
	content, chosenLang, err := legalres.Get(legalres.KeyTermsOfService, lang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 1,
			"msg":  "failed to load terms_of_service",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": legalDocData{
			Key:         legalres.KeyTermsOfService,
			Lang:        chosenLang,
			Format:      "markdown",
			Title:       legalres.ExtractTitle(content),
			LastUpdated: legalres.ExtractLastUpdated(content),
			Content:     content,
		},
	})
}

func GetPrivacyPolicy(c *gin.Context) {
	lang := legalres.ParsePreferredLang(c.GetHeader("Accept-Language"), c.Query("lang"))
	content, chosenLang, err := legalres.Get(legalres.KeyPrivacyPolicy, lang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 1,
			"msg":  "failed to load privacy_policy",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": legalDocData{
			Key:         legalres.KeyPrivacyPolicy,
			Lang:        chosenLang,
			Format:      "markdown",
			Title:       legalres.ExtractTitle(content),
			LastUpdated: legalres.ExtractLastUpdated(content),
			Content:     content,
		},
	})
}
