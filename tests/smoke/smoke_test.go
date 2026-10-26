package smoke

import "testing"

// TestModuleSkeleton は製品の動作を検証しない。
// Go 1.27 は、パッケージが無いモジュールで go test ./... を失敗させる。
func TestModuleSkeleton(t *testing.T) {}
