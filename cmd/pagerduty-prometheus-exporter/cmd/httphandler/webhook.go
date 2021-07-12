package httphandler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/24el/pagerduty-prometheus-exporter/internal/pagerduty"
)

const signatureHeader = "X-PagerDuty-Signature"

var errInvalidSignature = errors.New("invalid signature")

type IncidentListener interface {
	IncidentEventTriggered(event pagerduty.WebhookV3Event) error
}

type WebhookHandler struct {
	logger           *zap.Logger
	incidentListener IncidentListener
	signatureSecret  []byte
}

func NewWebhookHandler(logger *zap.Logger, incidentListener IncidentListener, signatureSecret []byte) *WebhookHandler {
	return &WebhookHandler{
		logger:           logger,
		incidentListener: incidentListener,
		signatureSecret:  signatureSecret,
	}
}

func (h *WebhookHandler) InstallRoutes(r *mux.Router, incidentWebhookV3URL string) {
	r.Path(incidentWebhookV3URL).
		Methods(http.MethodPost).
		Name("incident_webhook").
		HandlerFunc(h.incidentWebhookV3)
}

func (h *WebhookHandler) incidentWebhookV3(w http.ResponseWriter, r *http.Request) {
	rb, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.logger.Debug("incident webhook v3 event received", zap.ByteString("body", rb))

	if err := h.verifySignature(rb, r); err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := r.Body.Close(); err != nil {
		h.logger.Error("body close error", zap.Error(err))
	}

	var webhookV3 pagerduty.WebhookV3

	if err := json.Unmarshal(rb, &webhookV3); err != nil {
		h.logger.Error("unmarshal webhook", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.incidentListener.IncidentEventTriggered(webhookV3.Event); err != nil {
		h.logger.Error("handle webhook", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) verifySignature(reqPayload []byte, req *http.Request) error {
	if h.signatureSecret == nil {
		return nil
	}

	mac := hmac.New(sha256.New, h.signatureSecret)
	if _, err := mac.Write(reqPayload); err != nil {
		return err
	}

	expSignature := fmt.Sprintf("v1=%s", hex.EncodeToString(mac.Sum(nil)))

	signature := req.Header.Get(signatureHeader)
	if signature == "" {
		return errInvalidSignature
	}

	signatures := strings.Split(signature, ",")

	for _, sign := range signatures {
		if sign == expSignature {
			return nil
		}
	}

	h.logger.Debug(
		"incident webhook v3 sign verify failed",
		zap.String("expected_signature", expSignature),
		zap.String("signature", signature),
	)

	return errInvalidSignature
}
