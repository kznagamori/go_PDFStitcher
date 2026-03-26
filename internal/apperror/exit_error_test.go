package apperror_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
)

func TestExitErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"ExitOK", apperror.ExitOK, 0},
		{"ExitGeneral", apperror.ExitGeneral, 1},
		{"ExitInputFile", apperror.ExitInputFile, 2},
		{"ExitOutputFile", apperror.ExitOutputFile, 3},
		{"ExitPassword", apperror.ExitPassword, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestNewGeneralError(t *testing.T) {
	cause := fmt.Errorf("underlying cause")
	err := apperror.NewGeneralError("general error message", cause)

	if err.Code != apperror.ExitGeneral {
		t.Errorf("Code = %d, want %d", err.Code, apperror.ExitGeneral)
	}
	if err.Message != "general error message" {
		t.Errorf("Message = %q, want %q", err.Message, "general error message")
	}
	if err.Err != cause {
		t.Errorf("Err = %v, want %v", err.Err, cause)
	}
}

func TestNewInputFileError(t *testing.T) {
	cause := fmt.Errorf("file not found")
	err := apperror.NewInputFileError("input file not found: test.pdf", cause)

	if err.Code != apperror.ExitInputFile {
		t.Errorf("Code = %d, want %d", err.Code, apperror.ExitInputFile)
	}
	if err.Message != "input file not found: test.pdf" {
		t.Errorf("Message = %q, want %q", err.Message, "input file not found: test.pdf")
	}
}

func TestNewOutputFileError(t *testing.T) {
	cause := fmt.Errorf("permission denied")
	err := apperror.NewOutputFileError("cannot write to output path: permission denied", cause)

	if err.Code != apperror.ExitOutputFile {
		t.Errorf("Code = %d, want %d", err.Code, apperror.ExitOutputFile)
	}
}

func TestNewPasswordError(t *testing.T) {
	err := apperror.NewPasswordError("incorrect password for: secret.pdf", nil)

	if err.Code != apperror.ExitPassword {
		t.Errorf("Code = %d, want %d", err.Code, apperror.ExitPassword)
	}
}

func TestExitError_Error(t *testing.T) {
	err := apperror.NewGeneralError("test message", nil)
	if err.Error() != "test message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test message")
	}
}

func TestExitError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := apperror.NewInputFileError("wrapper message", cause)

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestExitError_UnwrapNil(t *testing.T) {
	// 原因エラーがnilの場合もUnwrapが正常動作することを確認
	err := apperror.NewPasswordError("no cause", nil)

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

func TestExitError_ErrorsAs(t *testing.T) {
	cause := fmt.Errorf("original error")
	exitErr := apperror.NewInputFileError("input error", cause)

	// errors.As で ExitError を正しく判定できることを確認
	var target *apperror.ExitError
	if !errors.As(exitErr, &target) {
		t.Fatal("errors.As が *apperror.ExitError にマッチしなかった")
	}
	if target.Code != apperror.ExitInputFile {
		t.Errorf("Code = %d, want %d", target.Code, apperror.ExitInputFile)
	}
}

func TestExitError_ErrorsIs(t *testing.T) {
	sentinel := fmt.Errorf("sentinel error")
	exitErr := apperror.NewGeneralError("wrapped sentinel", sentinel)

	// ラップされたセンチネルエラーを errors.Is で検出できることを確認
	if !errors.Is(exitErr, sentinel) {
		t.Fatal("errors.Is がラップされたセンチネルエラーを検出できなかった")
	}
}

func TestExitError_ImplementsErrorInterface(t *testing.T) {
	// ExitError が error インターフェースを満たすことを確認
	var err error = apperror.NewGeneralError("test", nil)
	if err == nil {
		t.Fatal("ExitError は error インターフェースを実装すべき")
	}
}
