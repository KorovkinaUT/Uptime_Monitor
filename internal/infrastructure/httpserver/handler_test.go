package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/application"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

type fakeMonitorService struct {
	created application.MonitorWithLastCheck
	getErr  error
}

func (f *fakeMonitorService) Create(context.Context, domain.Monitor) (application.MonitorWithLastCheck, error) {
	return f.created, nil
}

func (f *fakeMonitorService) Get(context.Context, uuid.UUID) (application.MonitorWithLastCheck, error) {
	return f.created, f.getErr
}

func (f *fakeMonitorService) List(context.Context) ([]application.MonitorWithLastCheck, error) {
	return []application.MonitorWithLastCheck{f.created}, nil
}

func (f *fakeMonitorService) Update(context.Context, domain.Monitor) (application.MonitorWithLastCheck, error) {
	return f.created, nil
}

func (f *fakeMonitorService) Delete(context.Context, uuid.UUID) error {
	return nil
}

type fakeCheckService struct{}

func (fakeCheckService) Check(context.Context, uuid.UUID) (domain.Check, error) {
	return domain.Check{}, nil
}

func (fakeCheckService) ListHistory(context.Context, uuid.UUID, int) ([]domain.Check, error) {
	return []domain.Check{}, nil
}

func TestCreateInvalidJSON(t *testing.T) {
	//arrange
	service := &fakeMonitorService{}
	router := NewHandler(Dependencies{Monitors: service, Checks: fakeCheckService{}, WebDir: t.TempDir()})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/monitors", strings.NewReader(`{"name":`))
	recorder := httptest.NewRecorder()

	//act
	router.ServeHTTP(recorder, request)

	//assert
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestGetInvalidUUID(t *testing.T) {
	//arrange
	router := NewHandler(Dependencies{Monitors: &fakeMonitorService{}, Checks: fakeCheckService{}, WebDir: t.TempDir()})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/not-a-uuid", nil)
	recorder := httptest.NewRecorder()

	//act
	router.ServeHTTP(recorder, request)

	//assert
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestGetNotFound(t *testing.T) {
	//arrange
	service := &fakeMonitorService{getErr: domain.ErrMonitorNotFound}
	router := NewHandler(Dependencies{Monitors: service, Checks: fakeCheckService{}, WebDir: t.TempDir()})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/monitors"+uuid.NewString()+"/", nil)
	recorder := httptest.NewRecorder()

	//act
	router.ServeHTTP(recorder, request)

	//assert
	if recorder.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestCreateValidRequest(t *testing.T) {
	//arrange
	monitorID := uuid.New()
	service := &fakeMonitorService{created: application.MonitorWithLastCheck{
		Monitor: domain.Monitor{
			ID:              monitorID,
			Name:            "My API",
			URL:             "https://example.com/health",
			IntervalSeconds: 30,
			TimeoutSeconds:  5,
			Enabled:         true,
		},
	}}
	router := NewHandler(Dependencies{Monitors: service, Checks: fakeCheckService{}, WebDir: t.TempDir()})
	body := `{"name":"My API","url":"https://example.com/health","interval_seconds":30,"timeout_seconds":5,"enabled":true}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/monitors", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	//act
	router.ServeHTTP(recorder, request)

	//assert
	if recorder.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if !strings.Contains(recorder.Body.String(), `"status":"unknown"`) {
		t.Errorf("body = %s, want unknown status", recorder.Body.String())
	}
}

func TestWriteErrorHidesInternalDetails(t *testing.T) {
	//arrange
	recorder := httptest.NewRecorder()
	logger := newTestLogger()

	//act
	writeError(recorder, logger, errors.New("database password leaked"))

	//assert
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if strings.Contains(recorder.Body.String(), "database password") {
		t.Errorf("body = %s, must hide internal error", recorder.Body.String())
	}
}
