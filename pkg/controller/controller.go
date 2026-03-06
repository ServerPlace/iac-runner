package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ServerPlace/iac-controller/pkg/api"
	"github.com/ServerPlace/iac-runner/pkg/cloudauth"
	"github.com/ServerPlace/iac-runner/pkg/log"
	"net/http"
	"time"
)

type Client interface {
	AccessToken(ctx context.Context, request api.CredentialsRequest) (*api.CredentialsResponse, error)
	RegisterPlan(ctx context.Context, request api.RegisterPlanRequest) (*api.RegisterPlanResponse, error)
	ClosePlan(ctx context.Context, request api.ClosePlanRequest) (*api.ClosePlanResponse, error)
}
type CtrlClient struct {
	client *cloudauth.Client
	signer *SignerRemote
}

const (
	credentialEndpoint string = "/v1/credentials"
	plansEndpoint      string = "/api/v1/plans"
	closePlanEndpoint  string = "/api/v1/plans/close"
)

func NewController(ctx context.Context, serviceUrl string, secretKey string) (*CtrlClient, error) {
	logger := log.FromContext(ctx)
	auth, err := cloudauth.NewCloudRunClient(ctx, serviceUrl)
	if err != nil { /* handle */
		logger.Err(err).Msgf("Could not connect to controller %s", serviceUrl)
		return nil, err
	}
	s := &SignerRemote{secretKey: secretKey}
	return &CtrlClient{
		client: auth,
		signer: s,
	}, nil
}

func (c *CtrlClient) AccessToken(ctx context.Context, request api.CredentialsRequest) (*api.CredentialsResponse, error) {
	logger := log.FromContext(ctx)

	var lastErr error
	maxRetries := 3
	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			logger.Info().
				Int("attempt", attempt).
				Int("max_retries", maxRetries).
				Dur("backoff", backoff).
				Msg("Retrying AccessToken request")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		request.Signature.Timestamp = time.Now().Unix()

		sig, err := c.signer.Sign(request)
		if err != nil {
			logger.Err(err).Msg("Failed to sign AccessToken request")
			return nil, err
		}
		request.Signature.Signature = sig

		reqBytes, err := json.Marshal(request)
		if err != nil {
			logger.Err(err).Msg("Failed to marshal CredentialsRequest")
			return nil, err
		}
		logger.Debug().Msgf("Request is %s", reqBytes)

		body := bytes.NewReader(reqBytes)
		req, err := http.NewRequest("POST", c.client.ServiceURL+credentialEndpoint, body)
		if err != nil {
			logger.Err(err).Msg("Failed to create AccessToken request")
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.DoWithTimeout(ctx, req, 300*time.Second)
		if err != nil {
			logger.Warn().Err(err).Int("attempt", attempt).Msg("Network error sending AccessToken")
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			logger.Error().Int("status", resp.StatusCode).Msg("Client error on AccessToken, not retrying")
			return nil, fmt.Errorf("controller responded with status code %d", resp.StatusCode)
		}

		if resp.StatusCode >= 500 {
			logger.Warn().Int("status", resp.StatusCode).Int("attempt", attempt).Msg("Server error on AccessToken, will retry")
			lastErr = fmt.Errorf("controller responded with status code %d", resp.StatusCode)
			continue
		}

		var token api.CredentialsResponse
		if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
			logger.Err(err).Msg("Failed to decode CredentialsResponse")
			return nil, fmt.Errorf("erro ao decodificar JSON access token: %s", err.Error())
		}

		return &token, nil
	}

	return nil, fmt.Errorf("AccessToken failed after %d attempts: %w", maxRetries, lastErr)
}

// RegisterPlan envia plan output para o backend com retry
func (c *CtrlClient) RegisterPlan(ctx context.Context, request api.RegisterPlanRequest) (*api.RegisterPlanResponse, error) {
	logger := log.FromContext(ctx)

	var lastErr error
	maxRetries := 3
	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			logger.Info().
				Int("attempt", attempt).
				Int("max_retries", maxRetries).
				Dur("backoff", backoff).
				Msg("Retrying RegisterPlan request")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		// Atualizar timestamp a cada tentativa (anti-replay protection)
		request.Signature.Timestamp = time.Now().Unix()

		sig, err := c.signer.Sign(request)
		if err != nil {
			logger.Err(err).Msg("Failed to sign request")
			return nil, err // Não vale retry em erro de crypto
		}
		request.Signature.Signature = sig

		reqBytes, err := json.Marshal(request)
		if err != nil {
			logger.Err(err).Msg("Failed to marshal RegisterPlanRequest")
			return nil, err // Não vale retry em erro de marshal
		}

		body := bytes.NewReader(reqBytes)
		req, err := http.NewRequest("POST", c.client.ServiceURL+plansEndpoint, body)
		if err != nil {
			logger.Err(err).Msg("Failed to create request")
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.DoWithTimeout(ctx, req, 30*time.Second)
		if err != nil {
			logger.Warn().Err(err).Int("attempt", attempt).Msg("Network error sending RegisterPlan")
			lastErr = err
			continue // Retry network errors
		}
		defer resp.Body.Close()

		// 4xx errors = não vale retry
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			logger.Error().Int("status", resp.StatusCode).Msg("Client error, not retrying")
			return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
		}

		// 5xx errors = vale retry
		if resp.StatusCode >= 500 {
			logger.Warn().Int("status", resp.StatusCode).Int("attempt", attempt).Msg("Server error, will retry")
			lastErr = fmt.Errorf("backend returned status %d", resp.StatusCode)
			continue
		}

		// Success (200/201)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			logger.Error().Int("status", resp.StatusCode).Msg("Unexpected status code")
			return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
		}

		var planResp api.RegisterPlanResponse
		if err := json.NewDecoder(resp.Body).Decode(&planResp); err != nil {
			logger.Err(err).Msg("Failed to decode response")
			return nil, err
		}

		logger.Info().
			Str("deployment_id", planResp.DeploymentID).
			Int("plan_version", planResp.PlanVersion).
			Msg("Plan registered successfully")

		return &planResp, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// ClosePlan notifica o backend que o apply foi concluído e fecha o PR/MR
func (c *CtrlClient) ClosePlan(ctx context.Context, request api.ClosePlanRequest) (*api.ClosePlanResponse, error) {
	logger := log.FromContext(ctx)

	var lastErr error
	maxRetries := 3
	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			logger.Info().
				Int("attempt", attempt).
				Int("max_retries", maxRetries).
				Dur("backoff", backoff).
				Msg("Retrying ClosePlan request")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		request.Signature.Timestamp = time.Now().Unix()

		sig, err := c.signer.Sign(request)
		if err != nil {
			logger.Err(err).Msg("Failed to sign ClosePlan request")
			return nil, err
		}
		request.Signature.Signature = sig

		reqBytes, err := json.Marshal(request)
		if err != nil {
			logger.Err(err).Msg("Failed to marshal ClosePlanRequest")
			return nil, err
		}

		body := bytes.NewReader(reqBytes)
		req, err := http.NewRequest("POST", c.client.ServiceURL+closePlanEndpoint, body)
		if err != nil {
			logger.Err(err).Msg("Failed to create ClosePlan request")
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.DoWithTimeout(ctx, req, 30*time.Second)
		if err != nil {
			logger.Warn().Err(err).Int("attempt", attempt).Msg("Network error sending ClosePlan")
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			logger.Error().Int("status", resp.StatusCode).Msg("Client error on ClosePlan, not retrying")
			return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
		}

		if resp.StatusCode >= 500 {
			logger.Warn().Int("status", resp.StatusCode).Int("attempt", attempt).Msg("Server error on ClosePlan, will retry")
			lastErr = fmt.Errorf("backend returned status %d", resp.StatusCode)
			continue
		}

		var closeResp api.ClosePlanResponse
		if err := json.NewDecoder(resp.Body).Decode(&closeResp); err != nil {
			logger.Err(err).Msg("Failed to decode ClosePlan response")
			return nil, err
		}

		logger.Info().
			Str("deployment_id", closeResp.DeploymentID).
			Str("status", closeResp.Status).
			Msg("Plan closed successfully")

		return &closeResp, nil
	}

	return nil, fmt.Errorf("ClosePlan failed after %d attempts: %w", maxRetries, lastErr)
}
