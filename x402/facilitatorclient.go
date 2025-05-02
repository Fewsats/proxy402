package x402

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DefaultFacilitatorURL is the default URL for the x402 facilitator service
const DefaultFacilitatorURL = "https://x402.org/facilitator"

// FacilitatorClient represents a facilitator client for verifying and settling payments
type FacilitatorClient struct {
	URL        string
	HTTPClient *http.Client
}

// NewFacilitatorClient creates a new facilitator client
func NewFacilitatorClient(url string) *FacilitatorClient {
	if url == "" {
		url = DefaultFacilitatorURL
	}
	return &FacilitatorClient{
		URL:        url,
		HTTPClient: http.DefaultClient,
	}
}

// Verify sends a payment verification request to the facilitator
func (c *FacilitatorClient) Verify(payload *PaymentPayload, requirements *PaymentRequirements) (*VerifyResponse, error) {
	reqBody := map[string]any{
		"x402Version":         1,
		"paymentPayload":      payload,
		"paymentRequirements": requirements,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Log the request body for debugging
	fmt.Printf("X402 Verify Request URL: %s/verify\n", c.URL)
	jsonBodyStr := string(jsonBody)
	if len(jsonBodyStr) > 1000 {
		fmt.Printf("X402 Verify Request Body (truncated): %s...\n", jsonBodyStr[:1000])
	} else {
		fmt.Printf("X402 Verify Request Body: %s\n", jsonBodyStr)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/verify", c.URL), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send verify request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response details
	fmt.Printf("X402 Verify Response Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("X402 Verify Response Headers: %v\n", resp.Header)

	respBodyStr := string(respBody)
	if len(respBodyStr) > 1000 {
		fmt.Printf("X402 Verify Response Body (truncated): %s...\n", respBodyStr[:1000])
	} else {
		fmt.Printf("X402 Verify Response Body: %s\n", respBodyStr)
	}

	// Create a new io.ReadCloser from the read body for further processing
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to verify payment: %s, response: %s", resp.Status, respBodyStr)
	}

	var verifyResp VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return nil, fmt.Errorf("failed to decode verify response: %w, raw response: %s", err, respBodyStr)
	}

	return &verifyResp, nil
}

// Settle sends a payment settlement request to the facilitator
func (c *FacilitatorClient) Settle(payload *PaymentPayload, requirements *PaymentRequirements) (*SettleResponse, error) {
	reqBody := map[string]any{
		"x402Version":         1,
		"paymentPayload":      payload,
		"paymentRequirements": requirements,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Log the request body for debugging
	fmt.Printf("X402 Settle Request URL: %s/settle\n", c.URL)
	jsonBodyStr := string(jsonBody)
	if len(jsonBodyStr) > 1000 {
		fmt.Printf("X402 Settle Request Body (truncated): %s...\n", jsonBodyStr[:1000])
	} else {
		fmt.Printf("X402 Settle Request Body: %s\n", jsonBodyStr)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/settle", c.URL), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send settle request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response details
	fmt.Printf("X402 Settle Response Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("X402 Settle Response Headers: %v\n", resp.Header)

	respBodyStr := string(respBody)
	if len(respBodyStr) > 1000 {
		fmt.Printf("X402 Settle Response Body (truncated): %s...\n", respBodyStr[:1000])
	} else {
		fmt.Printf("X402 Settle Response Body: %s\n", respBodyStr)
	}

	// Create a new io.ReadCloser from the read body for further processing
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to settle payment: %s, response: %s", resp.Status, respBodyStr)
	}

	var settleResp SettleResponse
	if err := json.NewDecoder(resp.Body).Decode(&settleResp); err != nil {
		return nil, fmt.Errorf("failed to decode settle response: %w, raw response: %s", err, respBodyStr)
	}

	return &settleResp, nil
}
