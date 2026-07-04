package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"project-prism/server/internal/models"
	"project-prism/server/internal/store"
)

type fakeBranchStore struct {
	createFn  func(context.Context, store.CreateBranchParams) (*models.Branch, error)
	listFn    func(context.Context, string) ([]*models.Branch, error)
	saveAllFn func(context.Context, string, []store.SaveBranchParams) ([]*models.Branch, error)
	getFn     func(context.Context, string, string) (*models.Branch, error)
	updateFn  func(context.Context, string, string, store.UpdateBranchParams) (*models.Branch, error)
	deleteFn  func(context.Context, string, string) error
}

func (f *fakeBranchStore) Create(ctx context.Context, p store.CreateBranchParams) (*models.Branch, error) {
	return f.createFn(ctx, p)
}
func (f *fakeBranchStore) ListByExperimentID(ctx context.Context, experimentID string) ([]*models.Branch, error) {
	if f.listFn == nil {
		return nil, nil
	}
	return f.listFn(ctx, experimentID)
}
func (f *fakeBranchStore) SaveAll(ctx context.Context, experimentID string, branches []store.SaveBranchParams) ([]*models.Branch, error) {
	return f.saveAllFn(ctx, experimentID, branches)
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
	longBranchName := strings.Repeat("a", branchNameMaxLength+1)
	longBranchKey := strings.Repeat("k", branchKeyMaxLength+1)
	largeBranchMetadata := `{"value":"` + strings.Repeat("a", branchMetadataMaxBytes) + `"}`

	t.Run("success", func(t *testing.T) {
		var got store.CreateBranchParams
		handler := NewBranchHandler(&fakeBranchStore{
			createFn: func(_ context.Context, p store.CreateBranchParams) (*models.Branch, error) {
				got = p
				return &models.Branch{ID: "branch_123", Key: p.Key, Name: p.Name}, nil
			},
			listFn: func(context.Context, string) ([]*models.Branch, error) {
				return []*models.Branch{{ID: "branch_existing", Weight: 50}}, nil
			},
		}, &fakeBranchExperimentStore{
			getFn: func(context.Context, string, string) (*models.Experiment, error) {
				return &models.Experiment{ID: "exp_123"}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPost, "/", `{"key":" control ","name":" Control ","weight":50}`, map[string]string{
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
		listErr    error
		branches   []*models.Branch
		verifyErr  error
		createErr  error
		wantStatus int
	}{
		{"experiment missing", `{"key":"control","name":"Control","weight":50}`, nil, nil, store.ErrNotFound, nil, http.StatusNotFound},
		{"verify error", `{"key":"control","name":"Control","weight":50}`, nil, nil, errors.New("boom"), nil, http.StatusInternalServerError},
		{"invalid body", "{", nil, nil, nil, nil, http.StatusBadRequest},
		{"key too long", `{"key":"` + longBranchKey + `","name":"Control","weight":50}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"name too long", `{"key":"control","name":"` + longBranchName + `","weight":50}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"metadata must be object", `{"key":"control","name":"Control","weight":50,"metadata_json":[]}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"metadata too large", `{"key":"control","name":"Control","weight":50,"metadata_json":` + largeBranchMetadata + `}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"missing key", `{"name":"Control","weight":50}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"missing name", `{"key":"control","weight":50}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"invalid weight", `{"key":"control","name":"Control","weight":200}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"list error", `{"key":"control","name":"Control","weight":50}`, errors.New("boom"), nil, nil, nil, http.StatusInternalServerError},
		{"invalid total", `{"key":"control","name":"Control","weight":50}`, nil, []*models.Branch{{ID: "branch_existing", Weight: 40}}, nil, nil, http.StatusUnprocessableEntity},
		{"conflict", `{"key":"control","name":"Control","weight":50}`, nil, []*models.Branch{{ID: "branch_existing", Weight: 50}}, nil, store.ErrConflict, http.StatusConflict},
		{"create error", `{"key":"control","name":"Control","weight":50}`, nil, []*models.Branch{{ID: "branch_existing", Weight: 50}}, nil, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewBranchHandler(&fakeBranchStore{
				createFn: func(context.Context, store.CreateBranchParams) (*models.Branch, error) {
					return nil, tc.createErr
				},
				listFn: func(context.Context, string) ([]*models.Branch, error) {
					if tc.listErr != nil {
						return nil, tc.listErr
					}
					return tc.branches, nil
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
	longBranchName := strings.Repeat("a", branchNameMaxLength+1)
	largeBranchMetadata := `{"value":"` + strings.Repeat("a", branchMetadataMaxBytes) + `"}`

	t.Run("success", func(t *testing.T) {
		var got store.UpdateBranchParams
		handler := NewBranchHandler(&fakeBranchStore{
			updateFn: func(_ context.Context, experimentID, id string, p store.UpdateBranchParams) (*models.Branch, error) {
				got = p
				return &models.Branch{ID: id, Name: p.Name, Weight: p.Weight}, nil
			},
			listFn: func(context.Context, string) ([]*models.Branch, error) {
				return []*models.Branch{
					{ID: "branch_123", Weight: 50},
					{ID: "branch_456", Weight: 30},
				}, nil
			},
		}, &fakeBranchExperimentStore{
			getFn: func(context.Context, string, string) (*models.Experiment, error) {
				return &models.Experiment{ID: "exp_123"}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPut, "/", `{"name":" Variant A ","weight":70}`, map[string]string{
			"appID":        "app_123",
			"experimentID": "exp_123",
			"id":           "branch_123",
		})

		handler.Update(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if got.Name != "Variant A" || got.Weight != 70 {
			t.Fatalf("unexpected params: %#v", got)
		}
	})

	testCases := []struct {
		name       string
		body       string
		listErr    error
		branches   []*models.Branch
		verifyErr  error
		updateErr  error
		wantStatus int
	}{
		{"experiment missing", `{"name":"Variant","weight":50}`, nil, nil, store.ErrNotFound, nil, http.StatusNotFound},
		{"verify error", `{"name":"Variant","weight":50}`, nil, nil, errors.New("boom"), nil, http.StatusInternalServerError},
		{"invalid body", "{", nil, nil, nil, nil, http.StatusBadRequest},
		{"name too long", `{"name":"` + longBranchName + `","weight":50}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"metadata must be object", `{"name":"Variant","weight":50,"metadata_json":[]}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"metadata too large", `{"name":"Variant","weight":50,"metadata_json":` + largeBranchMetadata + `}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"missing name", `{"name":" ","weight":50}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"invalid weight", `{"name":"Variant","weight":200}`, nil, nil, nil, nil, http.StatusUnprocessableEntity},
		{"list error", `{"name":"Variant","weight":50}`, errors.New("boom"), nil, nil, nil, http.StatusInternalServerError},
		{"branch missing", `{"name":"Variant","weight":50}`, nil, []*models.Branch{{ID: "branch_other", Weight: 100}}, nil, nil, http.StatusNotFound},
		{"invalid total", `{"name":"Variant","weight":50}`, nil, []*models.Branch{{ID: "branch_123", Weight: 40}, {ID: "branch_456", Weight: 30}}, nil, nil, http.StatusUnprocessableEntity},
		{"update error", `{"name":"Variant","weight":70}`, nil, []*models.Branch{{ID: "branch_123", Weight: 50}, {ID: "branch_456", Weight: 30}}, nil, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewBranchHandler(&fakeBranchStore{
				updateFn: func(context.Context, string, string, store.UpdateBranchParams) (*models.Branch, error) {
					return nil, tc.updateErr
				},
				listFn: func(context.Context, string) ([]*models.Branch, error) {
					if tc.listErr != nil {
						return nil, tc.listErr
					}
					return tc.branches, nil
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

func TestBranchHandlerSaveAll(t *testing.T) {
	longBranchName := strings.Repeat("a", branchNameMaxLength+1)
	longBranchKey := strings.Repeat("k", branchKeyMaxLength+1)
	largeBranchMetadata := `{"value":"` + strings.Repeat("a", branchMetadataMaxBytes) + `"}`

	t.Run("success", func(t *testing.T) {
		var got []store.SaveBranchParams
		handler := NewBranchHandler(&fakeBranchStore{
			saveAllFn: func(_ context.Context, experimentID string, branches []store.SaveBranchParams) ([]*models.Branch, error) {
				if experimentID != "exp_123" {
					t.Fatalf("unexpected experiment id: %s", experimentID)
				}
				got = branches
				return []*models.Branch{
					{ID: "branch_123", Key: "control", Name: "Control", Weight: 70},
					{ID: "branch_456", Key: "variant", Name: "Variant", Weight: 30},
				}, nil
			},
		}, &fakeBranchExperimentStore{
			getFn: func(context.Context, string, string) (*models.Experiment, error) {
				return &models.Experiment{ID: "exp_123"}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPut, "/", `{"branches":[{"id":" branch_123 ","key":" control ","name":" Control ","weight":70},{"key":" variant ","name":" Variant ","weight":30}]}`, map[string]string{
			"appID":        "app_123",
			"experimentID": "exp_123",
		})

		handler.SaveAll(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 branches, got %d", len(got))
		}
		if got[0].ID != "branch_123" || got[0].Key != "control" || got[0].Name != "Control" {
			t.Fatalf("unexpected first branch params: %#v", got[0])
		}
		if got[1].ID != "" || got[1].Key != "variant" || got[1].Name != "Variant" {
			t.Fatalf("unexpected second branch params: %#v", got[1])
		}
	})

	testCases := []struct {
		name       string
		body       string
		verifyErr  error
		saveErr    error
		wantStatus int
	}{
		{"experiment missing", `{"branches":[]}`, store.ErrNotFound, nil, http.StatusNotFound},
		{"verify error", `{"branches":[]}`, errors.New("boom"), nil, http.StatusInternalServerError},
		{"invalid body", "{", nil, nil, http.StatusBadRequest},
		{"key too long", `{"branches":[{"key":"` + longBranchKey + `","name":"Control","weight":100}]}`, nil, nil, http.StatusUnprocessableEntity},
		{"name too long", `{"branches":[{"key":"control","name":"` + longBranchName + `","weight":100}]}`, nil, nil, http.StatusUnprocessableEntity},
		{"metadata must be object", `{"branches":[{"key":"control","name":"Control","weight":100,"metadata_json":[]}]}`, nil, nil, http.StatusUnprocessableEntity},
		{"metadata too large", `{"branches":[{"key":"control","name":"Control","weight":100,"metadata_json":` + largeBranchMetadata + `}]}`, nil, nil, http.StatusUnprocessableEntity},
		{"missing key", `{"branches":[{"name":"Control","weight":100}]}`, nil, nil, http.StatusUnprocessableEntity},
		{"missing name", `{"branches":[{"key":"control","weight":100}]}`, nil, nil, http.StatusUnprocessableEntity},
		{"invalid weight", `{"branches":[{"key":"control","name":"Control","weight":200}]}`, nil, nil, http.StatusUnprocessableEntity},
		{"duplicate key", `{"branches":[{"key":"control","name":"Control","weight":50},{"key":"control","name":"Variant","weight":50}]}`, nil, nil, http.StatusConflict},
		{"duplicate id", `{"branches":[{"id":"branch_123","key":"control","name":"Control","weight":50},{"id":"branch_123","key":"variant","name":"Variant","weight":50}]}`, nil, nil, http.StatusUnprocessableEntity},
		{"invalid total", `{"branches":[{"key":"control","name":"Control","weight":40},{"key":"variant","name":"Variant","weight":30}]}`, nil, nil, http.StatusUnprocessableEntity},
		{"branch missing", `{"branches":[{"id":"branch_123","key":"control","name":"Control","weight":100}]}`, nil, store.ErrNotFound, http.StatusNotFound},
		{"conflict", `{"branches":[{"key":"control","name":"Control","weight":100}]}`, nil, store.ErrConflict, http.StatusConflict},
		{"save error", `{"branches":[{"key":"control","name":"Control","weight":100}]}`, nil, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewBranchHandler(&fakeBranchStore{
				saveAllFn: func(context.Context, string, []store.SaveBranchParams) ([]*models.Branch, error) {
					return nil, tc.saveErr
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
			req := newRequestWithURLParams(http.MethodPut, "/", tc.body, map[string]string{"appID": "app_123", "experimentID": "exp_123"})
			handler.SaveAll(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestBranchHandlerDelete(t *testing.T) {
	testCases := []struct {
		name       string
		listErr    error
		branches   []*models.Branch
		verifyErr  error
		deleteErr  error
		wantStatus int
	}{
		{"success", nil, []*models.Branch{{ID: "branch_123", Weight: 100}}, nil, nil, http.StatusNoContent},
		{"experiment missing", nil, nil, store.ErrNotFound, nil, http.StatusNotFound},
		{"verify error", nil, nil, errors.New("boom"), nil, http.StatusInternalServerError},
		{"list error", errors.New("boom"), nil, nil, nil, http.StatusInternalServerError},
		{"branch missing", nil, []*models.Branch{{ID: "branch_other", Weight: 100}}, nil, nil, http.StatusNotFound},
		{"invalid total", nil, []*models.Branch{{ID: "branch_123", Weight: 50}, {ID: "branch_456", Weight: 50}}, nil, nil, http.StatusUnprocessableEntity},
		{"delete error", nil, []*models.Branch{{ID: "branch_123", Weight: 100}}, nil, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewBranchHandler(&fakeBranchStore{
				deleteFn: func(context.Context, string, string) error { return tc.deleteErr },
				listFn: func(context.Context, string) ([]*models.Branch, error) {
					if tc.listErr != nil {
						return nil, tc.listErr
					}
					return tc.branches, nil
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
			req := newRequestWithURLParams(http.MethodDelete, "/", "", map[string]string{"appID": "app_123", "experimentID": "exp_123", "id": "branch_123"})
			handler.Delete(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}
