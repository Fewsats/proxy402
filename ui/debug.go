package ui

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// handleDebugPage renders the debug tool page
func (h *UIHandler) handleDebugPage(gCtx *gin.Context) {
	gCtx.HTML(http.StatusOK, "debug.html", gin.H{
		"baseURL":           h.getBaseURL(gCtx),
		"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
	})
}

// handleDebugTest processes and tests an x402 request
func (h *UIHandler) handleDebugTest(gCtx *gin.Context) {
	reqURL := gCtx.PostForm("url")
	method := gCtx.PostForm("method")
	paymentHeader := gCtx.PostForm("payment_header")

	if reqURL == "" {
		gCtx.HTML(http.StatusBadRequest, "debug_result.html", gin.H{
			"error": "URL is required",
		})
		return
	}

	// Create HTTP client and request
	client := &http.Client{}
	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "debug_result.html", gin.H{
			"error": "Failed to create request: " + err.Error(),
		})
		return
	}

	// Add payment header if provided
	if paymentHeader != "" {
		req.Header.Add("X-Payment", paymentHeader)
	}

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "debug_result.html", gin.H{
			"error": "Request failed: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "debug_result.html", gin.H{
			"error": "Failed to read response: " + err.Error(),
		})
		return
	}

	// Format headers for display
	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ", ")
	}

	// Send response data to template
	gCtx.HTML(http.StatusOK, "debug_result.html", gin.H{
		"status_code": resp.StatusCode,
		"status_text": http.StatusText(resp.StatusCode),
		"headers":     headers,
		"body":        string(bodyBytes),
	})
}