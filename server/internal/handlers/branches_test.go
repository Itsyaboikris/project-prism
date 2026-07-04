package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-prism/server/internal/models"
	"project-prism/server/internal/store"
)

type fakeBranchStore struct {
	createFn func(context.Context, store.CreateBranchParams) (*models.Branch, error)
	getFn    func(context.Context, string, string) (*models.Branch, error)
	updateFn func(context.Context, string, string, store.UpdateBranchParams) (*models.Branch, error)
	deleteFn func(context.Context, string, string) error
}

func (f *fakeBranchStore) Create(ctx context.Context, p store.CreateBranchParams) (*models.Branch, error) {
	return f.createFn(ctx, p)
}
func (f *fakeBranchStore) GetByID(ctx context.Context, experimentID, id string) (*models.Branch, error) {
	return f.getFn(ctx, experimentID, id)
}
func (f *fakeBranchStore) Update(ctx context.Context, experimentID, id string, p store.UpdateBranchParams) (*models.Branch, error) {
	return f.updateFn(ctx, experimentID, id, p)
}
func (f *fakeBranchStore) Delete(ctx context.Context, experimentID, id string) error {
	return f.deleteFn(ctx, experimentID, id)
}

type fakeBranchExperimentStore struct {
	getFn func(context.Context, string, string) (*models.Experiment, error)
}

func (f *fakeBranchExperimentStore) GetByID(ctx context.Context, applicationID, id string) (*models.Experiment, error) {
	return f.getFn(ctx, applicationID, id)
}

func TestBranchHandlerCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var got store.CreateBranchParams
		handler := NewBranchHandler(&fakeBranchStore{
			createFn: func(_ context.Context, p store.CreateBranchParams) (*models.Branch, error) {
				got = p
				return &models.Branch{ID: "branch_123", Key: p.Key, Name: p.Name}, nil
			},
		}, &fakeBranchExperimentStore{
			getFn: func(context.Context, string, string) (*models.Experiment, error) {
				return &models.Experiment{ID: "exp_123"}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPost, "/", `{"key":" control ","name":" Control ","weight":0.5}`, map[string]string{
			"appID":        "app_123",
			"experimentID": "exp_123",
		})

		handler.Create(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
		}
		if got.ExperimentID != "exp_123" || got.Key != "control" || got.Name != "Control" {
			t.Fatalf("unexpected params: %#v", got)
		}
	})

	testCases := []struct {
		name       string
		body       string
		verifyErr  error
		createErr  error
		wantStatus int
	}{
		{"experiment missing", `{"key":"control","name":"Control","weight":0.5}`, store.ErrNotFound, nil, http.StatusNotFound},
		{"verify error", `{"key":"control","name":"Control","weight":0.5}`, errors.New("boom"), nil, http.StatusInternalServerError},
		{"invalid body", "{", nil, nil, http.StatusBadRequest},
		{"missing key", `{"name":"Control","weight":0.5}`, nil, nil, http.StatusUnprocessableEntity},
		{"missing name", `{"key":"control","weight":0.5}`, nil, nil, http.StatusUnprocessableEntity},
		{"invalid weight", `{"key":"control","name":"Control","weight":2}`, nil, nil, http.StatusUnprocessableEntity},
		{"conflict", `{"key":"control","name":"Control","weight":0.5}`, nil, store.ErrConflict, http.StatusConflict},
		{"create error", `{"key":"control","name":"Control","weight":0.5}`, nil, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewBranchHandler(&fakeBranchStore{
				createFn: func(context.Context, store.CreateBranchParams) (*models.Branch, error) {
					return nil, tc.createErr
				},
			}, &fakeBranchExperimentStore{
				getFn: func(context.Context, string, string) (*models.Experiment, error) {
					if tc.verifyErr != nil {
						return nil, tc.verifyErr
					}
					return &models.Experiment{ID: "exp_123"}, nil
				},
			})
			rec := httptest.NewRecorder()
			req := newRequestWithURLParams(http.MethodPost, "/", tc.body, map[string]string{"appID": "app_123", "experimentID": "exp_123"})
			handler.Create(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestBranchHandlerUpdate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var got store.UpdateBranchParams
		handler := NewBranchHandler(&fakeBranchStore{
			updateFn: func(_ context.Context, experimentID, id string, p store.UpdateBranchParams) (*models.Branch, error) {
				got = p
				return &models.Branch{ID: id, Name: p.Name, Weight: p.Weight}, nil
			},
		}, &fakeBranchExperimentStore{
			getFn: func(context.Context, string, string) (*models.Experiment, error) {
				return &models.Experiment{ID: "exp_123"}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPut, "/", `{"name":" Variant A ","weight":0.7}`, map[string]string{
			"appID":        "app_123",
			"experimentID": "exp_123",
			"id":           "branch_123",
		})

		handler.Update(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if got.Name != "Variant A" || got.Weight != 0.7 {
			t.Fatalf("unexpected params: %#v", got)
		}
	})

	testCases := []struct {
		name       string
		body       string
		verifyErr  error
		updateErr  error
		wantStatus int
	}{
		{"experiment missing", `{"name":"Variant","weight":0.5}`, store.ErrNotFound, nil, http.StatusNotFound},
		{"verify error", `{"name":"Variant","weight":0.5}`, errors.New("boom"), nil, http.StatusInternalServerError},
		{"invalid body", "{", nil, nil, http.StatusBadRequest},
		{"missing name", `{"name":" ","weight":0.5}`, nil, nil, http.StatusUnprocessableEntity},
		{"invalid weight", `{"name":"Variant","weight":2}`, nil, nil, http.StatusUnprocessableEntity},
		{"branch missing", `{"name":"Variant","weight":0.5}`, nil, store.ErrNotFound, http.StatusNotFound},
		{"update error", `{"name":"Variant","weight":0.5}`, nil, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewBranchHandler(&fakeBranchStore{
				updateFn: func(context.Context, string, string, store.UpdateBranchParams) (*models.Branch, error) {
					return nil, tc.updateErr
				},
			}, &fakeBranchExperimentStore{
				getFn: func(context.Context, string, string) (*models.Experiment, error) {
					if tc.verifyErr != nil {
						return nil, tc.verifyErr
					}
					return &models.Experiment{ID: "exp_123"}, nil
				},
			})
			rec := httptest.NewRecorder()
			req := newRequestWithURLParams(http.MethodPut, "/", tc.body, map[string]string{"appID": "app_123", "experimentID": "exp_123", "id": "branch_123"})
			handler.Update(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestBranchHandlerDelete(t *testing.T) {
	testCases := []struct {
		name       string
		verifyErr  error
		deleteErr  error
		wantStatus int
	}{
		{"success", nil, nil, http.StatusNoContent},
		{"experiment missing", store.ErrNotFound, nil, http.StatusNotFound},
		{"verify error", errors.New("boom"), nil, http.StatusInternalServerError},
		{"branch missing", nil, store.ErrNotFound, http.StatusNotFound},
		{"delete error", nil, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewBranchHandler(&fakeBranchStore{
				deleteFn: func(context.Context, string, string) error { return tc.deleteErr },
			}, &fakeBranchExperimentStore{
				getFn: func(context.Context, string, string) (*models.Experiment, error) {
					if tc.verifyErr != nil {
						return nil, tc.verifyErr
					}
					return &models.Experiment{ID: "exp_123"}, nil
				},
			})
			rec := httptest.NewRecorder()
			req := newRequestWithURLParams(http.MethodDelete, "/", "", map[string]string{"appID": "app_123", "experimentID": "exp_123", "id": "branch_123"})
			handler.Delete(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}
