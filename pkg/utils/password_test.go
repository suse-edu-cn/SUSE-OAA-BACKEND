package utils

import "testing"

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		wantErr string
	}{
		{name: "合法密码", pass: "pass1234", wantErr: ""},
		{name: "合法包含特殊符号", pass: "p@ssword123", wantErr: ""},
		{name: "少于8位", pass: "pass123", wantErr: "密码长度不能少于8位"},
		{name: "纯数字", pass: "12345678", wantErr: "密码必须包含字母和数字"},
		{name: "纯字母", pass: "abcdefgh", wantErr: "密码必须包含字母和数字"},
		{name: "空密码", pass: "", wantErr: "密码长度不能少于8位"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.pass)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
			} else {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("expected error %q, got %v", tc.wantErr, err)
				}
			}
		})
	}
}
