package auth

import (
	"testing"
)

func TestParseQstubURLParams(t *testing.T) {
	m := &UnifiedAuthMiddleware{}

	tests := []struct {
		name      string
		authHeader string
		wantUID   string
		wantUtype uint32
		wantEmail string
		wantErr   bool
	}{
		{
			name:       "基本格式 - uid 和 ut",
			authHeader: "QiniuStub uid=1369077332&ut=1",
			wantUID:    "1369077332",
			wantUtype:  1,
			wantErr:    false,
		},
		{
			name:       "仅 uid",
			authHeader: "QiniuStub uid=12345",
			wantUID:    "12345",
			wantUtype:  0,
			wantErr:    false,
		},
		{
			name:       "完整格式",
			authHeader: "QiniuStub uid=1369077332&ut=1&app=1&email=test@qiniu.com",
			wantUID:    "1369077332",
			wantUtype:  1,
			wantEmail:  "test@qiniu.com",
			wantErr:    false,
		},
		{
			name:       "缺少 uid - 错误",
			authHeader: "QiniuStub ut=1",
			wantErr:    true,
		},
		{
			name:       "空参数 - 错误",
			authHeader: "QiniuStub ",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userInfo, err := m.parseQstubURLParams(tt.authHeader)

			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误，但成功了")
				}
				return
			}

			if err != nil {
				t.Errorf("不期望错误，但失败了: %v", err)
				return
			}

			if userInfo.UID != tt.wantUID {
				t.Errorf("UID = %v, want %v", userInfo.UID, tt.wantUID)
			}

			if userInfo.Utype != tt.wantUtype {
				t.Errorf("Utype = %v, want %v", userInfo.Utype, tt.wantUtype)
			}

			if tt.wantEmail != "" && userInfo.Email != tt.wantEmail {
				t.Errorf("Email = %v, want %v", userInfo.Email, tt.wantEmail)
			}
		})
	}
}

func TestParseQstubToken(t *testing.T) {
	m := &UnifiedAuthMiddleware{}

	// 测试 URL 参数格式
	t.Run("URL 参数格式", func(t *testing.T) {
		authHeader := "QiniuStub uid=1369077332&ut=1"
		userInfo, err := m.parseQstubToken(authHeader)

		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}

		if userInfo.UID != "1369077332" {
			t.Errorf("UID = %v, want 1369077332", userInfo.UID)
		}

		if userInfo.Utype != 1 {
			t.Errorf("Utype = %v, want 1", userInfo.Utype)
		}
	})

	// 测试错误格式
	t.Run("错误格式", func(t *testing.T) {
		authHeader := "Bearer some_token"
		_, err := m.parseQstubToken(authHeader)

		if err == nil {
			t.Errorf("期望错误，但成功了")
		}
	})
}
