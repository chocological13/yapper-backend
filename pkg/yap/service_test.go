package yap

import (
	"context"
	"errors"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/chocological13/yapper-backend/pkg/users"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"mime/multipart"
	"testing"
	"time"

	"github.com/chocological13/yapper-backend/pkg/mocks"
	"github.com/stretchr/testify/mock"
)

var (
	validUUID = pgtype.UUID{
		Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Valid: true,
	}

	validTimestamp = pgtype.Timestamptz{
		Time:  time.Now(),
		Valid: true,
	}

	mockUser = &users.User{
		ID: validUUID,
	}
)

type testSetup struct {
	svc          YapService
	querier      *mocks.Querier
	userService  *mocks.UserService
	mediaService *mocks.MediaService
	db           *mocks.PgxPool
	tx           *mocks.PgxTx
}

func setupTest(t *testing.T) *testSetup {
	querier := mocks.NewQuerier(t)
	userService := mocks.NewUserService(t)
	mediaService := mocks.NewMediaService(t)
	db := mocks.NewPgxPool(t)
	tx := mocks.NewPgxTx(t)

	svc := NewYapService(db, querier, userService, mediaService)

	return &testSetup{
		svc:          svc,
		querier:      querier,
		userService:  userService,
		mediaService: mediaService,
		db:           db,
		tx:           tx,
	}
}

// Example test to verify setup
func TestYapService_Setup(t *testing.T) {
	setup := setupTest(t)
	if setup.svc == nil {
		t.Error("Expected service to be initialized")
	}
}

// !! This doesn't work pls halp 🥲
//func TestYapService_CreateYap(t *testing.T) {
//	tests := []struct {
//		name      string
//		req       CreateYapRequest
//		mockSetup func(*testSetup)
//		want      *YapResponse
//		wantErr   error
//	}{
//		{
//			name: "successful yap creation without media",
//			req: CreateYapRequest{
//				Content: "test content yap without media @mention #hashtag",
//			},
//			mockSetup: func(ts *testSetup) {
//				ctx := mock.Anything
//
//				// Mock getting current user
//				ts.userService.On("GetCurrentUser", ctx).
//					Return(mockUser, nil)
//
//				// Create mock transaction using mockery
//				mockTx := mocks.NewPgxTx(t)
//
//				// Mock transaction methods
//				mockTx.On("Commit", ctx).Return(nil)
//				mockTx.On("Rollback", ctx).Return(nil)
//
//				// Mock the QueryRow call that SQLC will make
//				mockTx.On("QueryRow",
//					ctx,
//					mock.AnythingOfType("string"),
//					mock.Anything, // These are the actual params that will be passed
//					mock.Anything,
//					mock.Anything,
//					mock.Anything,
//					mock.Anything,
//					mock.Anything,
//				).Return(newMockRow(repository.CreateYapRow{
//					YapID:     validUUID,
//					UserID:    validUUID,
//					Content:   "test content yap without media @mention #hashtag",
//					Hashtags:  []string{"hashtag"},
//					Mentions:  []string{"mention"},
//					CreatedAt: validTimestamp,
//					UpdatedAt: validTimestamp,
//				}))
//
//				// Mock Begin to return our mock transaction
//				ts.db.On("Begin", ctx).Return(mockTx, nil)
//
//				// Mock media service
//				mediaSvcTx := mocks.NewMediaService(t)
//				ts.mediaService.On("WithTx", mockTx).Return(mediaSvcTx)
//				mediaSvcTx.On("ListByContent", ctx, validUUID, YapContentType).
//					Return([]*repository.Medium{}, nil)
//			},
//			want: &YapResponse{
//				YapID:     validUUID,
//				UserID:    mockUser.ID,
//				Content:   "test content yap without media @mention #hashtag",
//				Hashtags:  []string{"hashtag"},
//				Mentions:  []string{"mention"},
//				Media:     []*MediaItem{},
//				CreatedAt: validTimestamp,
//				UpdatedAt: validTimestamp,
//			},
//			wantErr: nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			ts := setupTest(t)
//			tt.mockSetup(ts)
//
//			got, err := ts.svc.CreateYap(context.Background(), tt.req)
//
//			if tt.wantErr != nil {
//				assert.Error(t, err)
//				assert.Nil(t, got)
//			} else {
//				assert.NoError(t, err)
//				assert.Equal(t, tt.want.UserID, got.UserID)
//				assert.Equal(t, tt.want.Content, got.Content)
//				assert.Equal(t, tt.want.Hashtags, got.Hashtags)
//				assert.Equal(t, tt.want.Mentions, got.Mentions)
//				assert.Empty(t, got.Media)
//			}
//
//			ts.userService.AssertExpectations(t)
//			ts.querier.AssertExpectations(t)
//			ts.mediaService.AssertExpectations(t)
//			ts.db.AssertExpectations(t)
//			ts.tx.AssertExpectations(t)
//		})
//	}
//}

func TestYapService_UploadMedia(t *testing.T) {
	mockFile := &multipart.FileHeader{
		Filename: "test.jpg",
		Header:   make(map[string][]string),
	}
	mockFile.Header.Set("Content-Type", "image/jpeg")

	tests := []struct {
		name      string
		file      *multipart.FileHeader
		mockSetup func(*testSetup)
		want      *MediaItem
		wantErr   error
	}{
		{
			name: "successful media upload",
			file: mockFile,
			mockSetup: func(ts *testSetup) {
				// Mock getting current user
				ts.userService.On("GetCurrentUser", mock.Anything).
					Return(mockUser, nil)

				// Mock media service upload
				expectedFolder := fmt.Sprintf("yaps/%s", mockUser.ID)
				mockMediaResponse := &repository.Medium{
					MediaID: validUUID,
					Type:    "image",
					Url:     "https://storage.example.com/test.jpg",
				}
				ts.mediaService.On("UploadMedia", mock.Anything, mockFile, expectedFolder).
					Return(mockMediaResponse, nil)
			},
			want: &MediaItem{
				MediaID: validUUID,
				Type:    "image",
				URL:     "https://storage.example.com/test.jpg",
			},
			wantErr: nil,
		},
		{
			name: "unauthorized user",
			file: mockFile,
			mockSetup: func(ts *testSetup) {
				ts.userService.On("GetCurrentUser", mock.Anything).
					Return(nil, errors.New("unauthorized"))
			},
			want:    nil,
			wantErr: errors.New("unauthorized"),
		},
		{
			name: "media service upload failure",
			file: mockFile,
			mockSetup: func(ts *testSetup) {
				ts.userService.On("GetCurrentUser", mock.Anything).
					Return(mockUser, nil)

				ts.mediaService.On("UploadMedia", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("upload failed"))
			},
			want:    nil,
			wantErr: errors.New("upload failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := setupTest(t)
			tt.mockSetup(ts)

			got, err := ts.svc.UploadMedia(context.Background(), tt.file)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			ts.userService.AssertExpectations(t)
			ts.mediaService.AssertExpectations(t)
		})
	}
}

func TestYapService_GetYapByID(t *testing.T) {
	tests := []struct {
		name      string
		yapID     pgtype.UUID
		mockSetup func(*testSetup)
		want      *YapResponse
		wantErr   error
	}{
		{
			name:  "successful yap retrieval",
			yapID: validUUID,
			mockSetup: func(ts *testSetup) {
				mockYap := repository.GetYapByIDRow{
					YapID:     validUUID,
					UserID:    validUUID,
					Content:   "Test yap content",
					CreatedAt: validTimestamp,
					UpdatedAt: validTimestamp,
				}

				ts.querier.On("GetYapByID", mock.Anything, mock.Anything).Return(mockYap, nil)

				ts.mediaService.On("ListByContent", mock.Anything, mock.Anything, mock.Anything).Return([]*repository.Medium{}, nil)
			},
			want: &YapResponse{
				YapID:     validUUID,
				UserID:    validUUID,
				Content:   "Test yap content",
				Media:     []*MediaItem{},
				CreatedAt: validTimestamp,
				UpdatedAt: validTimestamp,
			},
			wantErr: nil,
		},
		{
			name:  "failed yap retrieval",
			yapID: validUUID,
			mockSetup: func(ts *testSetup) {
				ts.querier.On("GetYapByID", mock.Anything, mock.Anything).Return(repository.GetYapByIDRow{},
					pgx.ErrNoRows)
			},
			want:    nil,
			wantErr: ErrYapNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := setupTest(t)
			tt.mockSetup(ts)

			got, err := ts.svc.GetYapByID(context.Background(), tt.yapID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
				assert.Equal(t, tt.yapID, got.YapID)
				assert.Equal(t, tt.want.UserID, got.UserID)
				assert.Equal(t, tt.want.Content, got.Content)
				assert.Equal(t, tt.want.Media, got.Media)
			}

			ts.querier.AssertExpectations(t)
			ts.mediaService.AssertExpectations(t)
		})

	}
}

func TestYapService_ListYapsByUser(t *testing.T) {
	mockYaps := []repository.ListYapsByUserRow{
		{
			YapID:     validUUID,
			UserID:    validUUID,
			Content:   "Test yap content",
			CreatedAt: validTimestamp,
			UpdatedAt: validTimestamp,
		},
		{
			YapID:     validUUID,
			UserID:    validUUID,
			Content:   "Test yap content - 2",
			CreatedAt: validTimestamp,
			UpdatedAt: validTimestamp,
		},
	}

	expectedResponse := []*YapResponse{
		{
			YapID:     validUUID,
			UserID:    validUUID,
			Content:   "Test yap content",
			Media:     []*MediaItem{},
			CreatedAt: validTimestamp,
			UpdatedAt: validTimestamp,
		},
		{
			YapID:     validUUID,
			UserID:    validUUID,
			Content:   "Test yap content - 2",
			Media:     []*MediaItem{},
			CreatedAt: validTimestamp,
			UpdatedAt: validTimestamp,
		},
	}

	tests := []struct {
		name      string
		req       ListYapsRequest
		mockSetup func(*testSetup)
		want      []*YapResponse
		wantErr   error
	}{
		{
			name: "successful yap list by user retrieval",
			req: ListYapsRequest{
				UserID: validUUID.String(),
				Limit:  20,
				Offset: 0,
			},
			mockSetup: func(ts *testSetup) {
				ts.querier.On("ListYapsByUser", mock.Anything, repository.ListYapsByUserParams{
					UserID:  validUUID,
					Column2: int32(20),
					Column3: int32(0),
				}).Return(mockYaps, nil)

				ts.mediaService.On("ListByContent", mock.Anything, validUUID,
					YapContentType).Return([]*repository.Medium{}, nil)
			},
			want:    expectedResponse,
			wantErr: nil,
		},
		{
			name: "successful yap list for current user",
			req: ListYapsRequest{
				UserID: "",
				Limit:  20,
				Offset: 0,
			},
			mockSetup: func(ts *testSetup) {
				ts.userService.On("GetCurrentUser", mock.Anything).Return(&users.User{
					ID: validUUID,
				}, nil)

				ts.querier.On("ListYapsByUser", mock.Anything, repository.ListYapsByUserParams{
					UserID:  validUUID,
					Column2: int32(20),
					Column3: int32(0),
				}).Return(mockYaps, nil)

				ts.mediaService.On("ListByContent", mock.Anything, validUUID, YapContentType).Return([]*repository.Medium{}, nil)
			},
			want:    expectedResponse,
			wantErr: nil,
		},
		{
			name: "error getting current user",
			req: ListYapsRequest{
				UserID: "",
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(ts *testSetup) {
				ts.userService.On("GetCurrentUser", mock.Anything).
					Return(nil, apperrors.ErrContextNotFound)
			},
			want:    nil,
			wantErr: apperrors.ErrContextNotFound,
		},
		{
			name: "invalid UUID format",
			req: ListYapsRequest{
				UserID: "invalid-uuid",
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(ts *testSetup) {
				// No mocks needed - should fail before any service calls
			},
			want:    nil,
			wantErr: errors.New("invalid UUID format"),
		},
		{
			name: "database error",
			req: ListYapsRequest{
				UserID: validUUID.String(),
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(ts *testSetup) {
				ts.querier.On("ListYapsByUser", mock.Anything, mock.Anything).
					Return(nil, errors.New("database error"))
			},
			want:    nil,
			wantErr: errors.New("database error"),
		},
		{
			name: "no yaps found",
			req: ListYapsRequest{
				UserID: validUUID.String(),
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(ts *testSetup) {
				ts.querier.On("ListYapsByUser", mock.Anything, mock.Anything).
					Return([]repository.ListYapsByUserRow{}, pgx.ErrNoRows)
			},
			want:    nil,
			wantErr: ErrYapNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := setupTest(t)
			tt.mockSetup(ts)

			got, err := ts.svc.ListYapsByUser(context.Background(), tt.req)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
				assert.Equal(t, len(tt.want), len(got))
				assert.Equal(t, tt.want[0].Media, got[0].Media)
				assert.Equal(t, tt.want[0].YapID, got[0].YapID)
			}

			ts.querier.AssertExpectations(t)
			ts.mediaService.AssertExpectations(t)
		})
	}
}

// Helper to create a mock pgx.Row - this was an attempt at mocking a transaction query
//type mockRow struct {
//	repository.CreateYapRow
//}
//
//func (m *mockRow) Scan(dest ...interface{}) error {
//	// Map the fields to the destination pointers
//	yapRow := m.CreateYapRow
//	scanners := []interface{}{
//		&yapRow.YapID,
//		&yapRow.UserID,
//		&yapRow.Content,
//		&yapRow.Hashtags,
//		&yapRow.Mentions,
//		&yapRow.CreatedAt,
//		&yapRow.UpdatedAt,
//	}
//
//	for i, scanner := range scanners {
//		if i < len(dest) {
//			switch d := dest[i].(type) {
//			case *pgtype.UUID:
//				switch s := scanner.(type) {
//				case *pgtype.UUID:
//					*d = *s
//				}
//			case *string:
//				switch s := scanner.(type) {
//				case *string:
//					*d = *s
//				}
//			case *[]string:
//				switch s := scanner.(type) {
//				case *[]string:
//					*d = *s
//				}
//			case *time.Time:
//				switch s := scanner.(type) {
//				case *time.Time:
//					*d = *s
//				}
//			}
//		}
//	}
//	return nil
//}
//
//func newMockRow(row repository.CreateYapRow) pgx.Row {
//	return &mockRow{row}
//}
