package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	core2 "yadro.com/course/api/core"
	mock_core "yadro.com/course/api/core/mocks"
)

func TestNewUpdateStatsHandler(t *testing.T) {

	tests := []struct {
		name                 string
		mockBehavior         func(updater *mock_core.MockUpdater)
		expectedStatusCode   int
		expectedResponseBody string
	}{
		{
			name: "Success",
			mockBehavior: func(m *mock_core.MockUpdater) {
				m.EXPECT().Stats(gomock.Any()).Return(core2.UpdateStats{
					WordsTotal:    100,
					WordsUnique:   80,
					ComicsFetched: 50,
					ComicsTotal:   200,
				}, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedResponseBody: `{
				"words_total": 100,
				"words_unique": 80,
				"comics_fetched": 50,
				"comics_total": 200
			}`,
		},
		{
			name: "Service Error",
			mockBehavior: func(m *mock_core.MockUpdater) {
				m.EXPECT().Stats(gomock.Any()).Return(core2.UpdateStats{}, errors.New("database error"))
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedResponseBody: "database error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUpdater := mock_core.NewMockUpdater(ctrl)
			tt.mockBehavior(mockUpdater)

			logger := slog.Default()

			handler := NewUpdateStatsHandler(logger, mockUpdater)

			req := httptest.NewRequest(http.MethodGet, "/stats", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, tt.expectedStatusCode, w.Code)

			if tt.expectedStatusCode == http.StatusOK {
				var expected, actual map[string]interface{}
				assert.NoError(t, json.Unmarshal([]byte(tt.expectedResponseBody), &expected))
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))
				assert.Equal(t, expected, actual)

				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			} else {
				assert.Equal(t, tt.expectedResponseBody, w.Body.String())
			}
		})
	}
}

func TestNewUpdateStatusHandler(t *testing.T) {
	tests := []struct {
		name                 string
		mockBehavior         func(updater *mock_core.MockUpdater)
		expectedStatusCode   int
		expectedResponseBody string
	}{
		{
			name: "Success",
			mockBehavior: func(m *mock_core.MockUpdater) {
				m.EXPECT().Status(gomock.Any()).Return(core2.StatusUpdateRunning, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedResponseBody: `{
				"status": "running"
			}`,
		},
		{
			name: "Service Error",
			mockBehavior: func(m *mock_core.MockUpdater) {
				m.EXPECT().Status(gomock.Any()).Return(core2.StatusUpdateRunning, errors.New("database error"))
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedResponseBody: "database error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUpdater := mock_core.NewMockUpdater(ctrl)
			tt.mockBehavior(mockUpdater)

			logger := slog.Default()

			handler := NewUpdateStatusHandler(logger, mockUpdater)

			req := httptest.NewRequest(http.MethodGet, "/stats", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, tt.expectedStatusCode, w.Code)

			if tt.expectedStatusCode == http.StatusOK {
				var expected, actual map[string]interface{}
				assert.NoError(t, json.Unmarshal([]byte(tt.expectedResponseBody), &expected))
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))
				assert.Equal(t, expected, actual)

				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			} else {
				assert.Equal(t, tt.expectedResponseBody, w.Body.String())
			}
		})
	}
}
