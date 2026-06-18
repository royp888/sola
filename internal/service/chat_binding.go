package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

var errForbidden = errors.New("forbidden")

type chatBindingService struct{ store *store.Store }

func (s *chatBindingService) Bind(ctx context.Context, req bot.ChatBindingRequest) (*bot.ChatBinding, error) {
	if s.store == nil || s.store.DB == nil {
		return &bot.ChatBinding{ChatID: req.ChatID, ChatType: req.ChatType, Title: req.Title, Username: req.Username, InviteLink: req.InviteLink, BoundBy: req.BoundBy, Description: req.Description, BoundAt: time.Now()}, nil
	}
	var owner *model.User
	if req.OwnerTelegramUserID != 0 {
		displayName := strings.TrimSpace(req.OwnerDisplayName)
		if displayName == "" {
			displayName = req.OwnerUsername
		}
		user, err := upsertTelegramOwner(ctx, s.store, upsertTelegramOwnerInput{
			TelegramUserID: req.OwnerTelegramUserID,
			Username:       strings.TrimPrefix(req.OwnerUsername, "@"),
			DisplayName:    displayName,
			Role:           "owner",
		})
		if err != nil {
			return nil, err
		}
		owner = user
	}
	title := req.Title
	username := req.Username
	inviteLink := req.InviteLink
	description := req.Description
	var chat model.TelegramChat
	err := s.store.DB.WithContext(ctx).Where("telegram_chat_id = ?", req.ChatID).First(&chat).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil && owner != nil && chat.OwnerUserID != nil && *chat.OwnerUserID != owner.ID {
		return nil, fmt.Errorf("该群已被其他群主绑定")
	}
	updates := model.TelegramChat{
		TelegramChatID: req.ChatID,
		Type:           req.ChatType,
		Title:          &title,
		Username:       stringPtrOrNil(username),
		InviteLink:     stringPtrOrNil(inviteLink),
		Description:    stringPtrOrNil(description),
		Status:         "active",
		LastSeenAt:     timePtr(time.Now()),
	}
	if owner != nil {
		updates.OwnerUserID = &owner.ID
	}
	if err := s.store.DB.WithContext(ctx).Where("telegram_chat_id = ?", req.ChatID).Assign(updates).FirstOrCreate(&chat).Error; err != nil {
		return nil, err
	}
	if owner != nil {
		if err := s.ensureChatOwnerAdmin(ctx, chat, *owner); err != nil {
			return nil, err
		}
	}
	return chatBindingToBot(chat, req.BoundBy), nil
}

func (s *chatBindingService) Unbind(ctx context.Context, chatID int64) error {
	if s.store == nil || s.store.DB == nil {
		return nil
	}
	return s.store.DB.WithContext(ctx).Model(&model.TelegramChat{}).Where("telegram_chat_id = ?", chatID).Update("status", "inactive").Error
}

func (s *chatBindingService) List(ctx context.Context, query bot.CommonListQuery) ([]bot.ChatBinding, error) {
	if s.store == nil || s.store.DB == nil {
		return []bot.ChatBinding{}, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.TelegramChat{})
	if strings.TrimSpace(query.OwnerUserID) != "" {
		if ownerID, err := uuid.Parse(strings.TrimSpace(query.OwnerUserID)); err == nil {
			db = db.Where("owner_user_id = ?", ownerID)
		} else {
			return []bot.ChatBinding{}, nil
		}
	}
	var chats []model.TelegramChat
	if err := db.Order("created_at desc").Limit(normalLimit(query.Limit)).Offset(query.Offset).Find(&chats).Error; err != nil {
		return nil, err
	}
	out := make([]bot.ChatBinding, 0, len(chats))
	for _, chat := range chats {
		out = append(out, *chatBindingToBot(chat, ""))
	}
	return out, nil
}

func (s *chatBindingService) ListByTelegramUser(ctx context.Context, telegramUserID int64, limit int) ([]bot.ChatBinding, error) {
	if s.store == nil || s.store.DB == nil || telegramUserID == 0 {
		return []bot.ChatBinding{}, nil
	}
	var user model.User
	if err := s.store.DB.WithContext(ctx).Where("telegram_user_id = ?", telegramUserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []bot.ChatBinding{}, nil
		}
		return nil, err
	}
	return s.List(ctx, bot.CommonListQuery{OwnerUserID: user.ID.String(), Limit: limit})
}

func (s *chatBindingService) UserOwnsChat(ctx context.Context, userID string, chatID string) (bool, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return true, nil
	}
	ownerID, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return false, nil
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return true, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.TelegramChat{}).Where("owner_user_id = ?", ownerID)
	if tgChatID, err := strconv.ParseInt(chatID, 10, 64); err == nil {
		db = db.Where("telegram_chat_id = ?", tgChatID)
	} else if parsed, err := uuid.Parse(chatID); err == nil {
		db = db.Where("id = ?", parsed)
	} else {
		return false, nil
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *chatBindingService) ensureChatOwnerAdmin(ctx context.Context, chat model.TelegramChat, owner model.User) error {
	now := time.Now()
	admin := model.ChatAdmin{
		ChatID:          chat.ID,
		UserID:          owner.ID,
		Role:            "admin",
		CanManage:       true,
		CanPost:         true,
		CanDelete:       true,
		CanBan:          true,
		GrantedByUserID: &owner.ID,
		GrantedAt:       now,
	}
	return s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chat.ID, owner.ID).
		Assign(admin).
		FirstOrCreate(&admin).Error
}

func normalLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func timePtr(value time.Time) *time.Time { return &value }

type upsertTelegramOwnerInput struct {
	TelegramUserID int64
	Username       string
	DisplayName    string
	Role           string
	PhotoURL       string
}

func upsertTelegramOwner(ctx context.Context, st *store.Store, in upsertTelegramOwnerInput) (*model.User, error) {
	if st == nil || st.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	if in.TelegramUserID == 0 {
		return nil, errors.New("telegram user id is required")
	}
	role := strings.TrimSpace(in.Role)
	if role == "" {
		role = "owner"
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(in.Username)
	}
	now := time.Now()
	var user model.User
	err := st.DB.WithContext(ctx).Where("telegram_user_id = ?", in.TelegramUserID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		username := strings.TrimPrefix(strings.TrimSpace(in.Username), "@")
		user = model.User{
			BaseModel:      model.BaseModel{ID: uuid.New(), CreatedAt: now, UpdatedAt: now},
			TelegramUserID: &in.TelegramUserID,
			DisplayName:    displayName,
			Role:           role,
			LanguageCode:   "zh-CN",
			Timezone:       "Asia/Shanghai",
			Status:         "active",
			IsActive:       true,
			LastLoginAt:    &now,
		}
		if username != "" {
			user.Username = &username
		}
		if strings.TrimSpace(in.PhotoURL) != "" {
			user.MetadataJSON = fmt.Sprintf(`{"photo_url":%q}`, strings.TrimSpace(in.PhotoURL))
		}
		if user.MetadataJSON == "" {
			user.MetadataJSON = "{}"
		}
		if err := st.DB.WithContext(ctx).Create(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"display_name":  displayName,
		"last_login_at": now,
		"status":        "active",
		"is_active":     true,
		"updated_at":    now,
	}
	if strings.TrimSpace(user.Role) == "" || strings.TrimSpace(user.Role) == "user" {
		updates["role"] = role
		user.Role = role
	}
	if username := strings.TrimPrefix(strings.TrimSpace(in.Username), "@"); username != "" {
		updates["username"] = username
	}
	if err := st.DB.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := st.DB.WithContext(ctx).Where("telegram_user_id = ?", in.TelegramUserID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func ownedTelegramChatIDs(ctx context.Context, st *store.Store, ownerUserID string) ([]int64, error) {
	if strings.TrimSpace(ownerUserID) == "" {
		return nil, nil
	}
	if st == nil || st.DB == nil {
		return []int64{}, nil
	}
	ownerID, err := uuid.Parse(strings.TrimSpace(ownerUserID))
	if err != nil {
		return []int64{}, nil
	}
	var ids []int64
	err = st.DB.WithContext(ctx).
		Model(&model.TelegramChat{}).
		Where("owner_user_id = ?", ownerID).
		Pluck("telegram_chat_id", &ids).Error
	if isMissingTableError(err) {
		return []int64{}, nil
	}
	return ids, err
}

func ensureOwnedTelegramChatID(ctx context.Context, st *store.Store, ownerUserID string, chatID int64) error {
	if strings.TrimSpace(ownerUserID) == "" {
		return nil
	}
	if chatID == 0 {
		return errForbidden
	}
	ownedIDs, err := ownedTelegramChatIDs(ctx, st, ownerUserID)
	if err != nil {
		return err
	}
	if !containsTelegramChatID(ownedIDs, chatID) {
		return errForbidden
	}
	return nil
}

func containsTelegramChatID(ids []int64, chatID int64) bool {
	for _, id := range ids {
		if id == chatID {
			return true
		}
	}
	return false
}

func telegramChatIDInScope(chatID int64, requestedChatID int64, ownedIDs []int64) bool {
	if requestedChatID != 0 && chatID != requestedChatID {
		return false
	}
	if ownedIDs == nil {
		return true
	}
	if chatID == 0 {
		return false
	}
	return containsTelegramChatID(ownedIDs, chatID)
}

func scopeTelegramChatID(db *gorm.DB, column string, chatID int64, ids []int64) *gorm.DB {
	if chatID != 0 {
		if ids != nil && !containsTelegramChatID(ids, chatID) {
			return db.Where("1 = 0")
		}
		return db.Where(column+" = ?", chatID)
	}
	if ids != nil {
		if len(ids) == 0 {
			return db.Where("1 = 0")
		}
		return db.Where(column+" IN ?", ids)
	}
	return db
}

func chatBindingToBot(chat model.TelegramChat, boundBy string) *bot.ChatBinding {
	ownerUserID := ""
	if chat.OwnerUserID != nil {
		ownerUserID = chat.OwnerUserID.String()
	}
	return &bot.ChatBinding{
		ChatID:      chat.TelegramChatID,
		ChatType:    chat.Type,
		Title:       deref(chat.Title),
		Username:    deref(chat.Username),
		InviteLink:  deref(chat.InviteLink),
		BoundBy:     boundBy,
		Description: deref(chat.Description),
		OwnerUserID: ownerUserID,
		BoundAt:     chat.CreatedAt,
	}
}
