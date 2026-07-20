package service

import (
	"context"
	"fmt"
	"unicode/utf8"

	crmmodel "github.com/dever-package/crm/model"
	frontservice "github.com/dever-package/front/service"
)

const minimumWorkPasswordLength = 6

func (WorkService) ChangePassword(
	ctx context.Context,
	staff *WorkStaffSession,
	payload map[string]any,
) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	currentPassword := firstText(payload, "current_password", "currentPassword")
	newPassword := firstText(payload, "new_password", "newPassword")
	confirmPassword := firstText(payload, "confirm_password", "confirmPassword")
	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		return nil, fmt.Errorf("请完整填写原密码、新密码和确认密码")
	}
	if utf8.RuneCountInString(newPassword) < minimumWorkPasswordLength {
		return nil, fmt.Errorf("新密码至少需要 %d 位", minimumWorkPasswordLength)
	}
	if newPassword != confirmPassword {
		return nil, fmt.Errorf("两次输入的新密码不一致")
	}
	current := crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"id":     staff.ID,
		"status": crmmodel.StatusEnabled,
	})
	if current == nil || !frontservice.VerifyPassword(current.Password, currentPassword) {
		return nil, fmt.Errorf("原密码错误")
	}
	if newPassword == currentPassword {
		return nil, fmt.Errorf("新密码不能与原密码相同")
	}
	hashedPassword, err := frontservice.HashPassword(newPassword)
	if err != nil {
		return nil, fmt.Errorf("新密码处理失败")
	}
	if crmmodel.NewStaffModel().Update(ctx, map[string]any{
		"id":       current.ID,
		"status":   crmmodel.StatusEnabled,
		"password": current.Password,
	}, map[string]any{
		"password": hashedPassword,
	}) == 0 {
		return nil, fmt.Errorf("密码已发生变化，请重新登录后再试")
	}
	return map[string]any{"changed": true}, nil
}
