package prompt

import (
	"strings"
	"testing"
)

func TestGenerateDraft_Bugfix(t *testing.T) {
	draft := GenerateDraft("登录页面报错 500", TaskTypeBugfix)
	if !strings.Contains(draft, "任务目标") {
		t.Error("draft should contain 任务目标")
	}
	if !strings.Contains(draft, "登录页面报错 500") {
		t.Error("draft should contain original input")
	}
	if !strings.Contains(draft, "定位并修复") {
		t.Error("bugfix draft should contain 定位并修复")
	}
	if !strings.Contains(draft, "验收标准") {
		t.Error("draft should contain 验收标准")
	}
}

func TestGenerateDraft_Build(t *testing.T) {
	draft := GenerateDraft("添加用户注册功能", TaskTypeBuild)
	if !strings.Contains(draft, "实现以下功能") {
		t.Error("build draft should contain 实现以下功能")
	}
	if !strings.Contains(draft, "添加用户注册功能") {
		t.Error("draft should contain original input")
	}
}

func TestGenerateDraft_Review(t *testing.T) {
	draft := GenerateDraft("审查 PR #123", TaskTypeReview)
	if !strings.Contains(draft, "审查以下代码变更") {
		t.Error("review draft should contain 审查以下代码变更")
	}
}

func TestGenerateDraft_QA(t *testing.T) {
	draft := GenerateDraft("测试首页加载", TaskTypeQA)
	if !strings.Contains(draft, "质量验证") {
		t.Error("qa draft should contain 质量验证")
	}
}

func TestGenerateDraft_Docs(t *testing.T) {
	draft := GenerateDraft("更新 README", TaskTypeDocs)
	if !strings.Contains(draft, "更新或创建以下文档") {
		t.Error("docs draft should contain 更新或创建以下文档")
	}
}

func TestGenerateDraft_Architecture(t *testing.T) {
	draft := GenerateDraft("设计微服务架构", TaskTypeArchitecture)
	if !strings.Contains(draft, "分析架构") {
		t.Error("architecture draft should contain 分析架构")
	}
}

func TestGenerateDraft_EmptyTaskType(t *testing.T) {
	// 空 taskType 应自动推断
	draft := GenerateDraft("修复登录 bug", "")
	if !strings.Contains(draft, "定位并修复") {
		t.Error("auto-inferred bugfix should contain 定位并修复")
	}
}

func TestGenerateDraft_UnknownTaskType(t *testing.T) {
	// 未知 taskType 应回退到 build
	draft := GenerateDraft("做点什么", "unknown_type")
	if !strings.Contains(draft, "实现以下功能") {
		t.Error("unknown type should fallback to build template")
	}
}

func TestInferTaskType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"修复登录 bug", TaskTypeBugfix},
		{"页面报错了", TaskTypeBugfix},
		{"测试首页", TaskTypeQA},
		{"检查 API 响应", TaskTypeQA},
		{"审查代码", TaskTypeReview},
		{"更新文档", TaskTypeDocs},
		{"设计新架构", TaskTypeArchitecture},
		{"添加用户功能", TaskTypeBuild},
		{"随便写点代码", TaskTypeBuild},
	}
	for _, tt := range tests {
		got := InferTaskType(tt.input)
		if got != tt.expected {
			t.Errorf("InferTaskType(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGetSupportedTaskTypes(t *testing.T) {
	types := GetSupportedTaskTypes()
	if len(types) != 6 {
		t.Errorf("expected 6 task types, got %d", len(types))
	}
	for _, expected := range []string{TaskTypeBugfix, TaskTypeBuild, TaskTypeReview, TaskTypeQA, TaskTypeDocs, TaskTypeArchitecture} {
		found := false
		for _, tt := range types {
			if tt == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing task type: %s", expected)
		}
	}
}

func TestGenerateDraft_AllSections(t *testing.T) {
	draft := GenerateDraft("test input", TaskTypeBuild)
	sections := []string{"## 任务目标", "## 上下文", "## 执行要求", "## 验收标准", "## 输出要求"}
	for _, s := range sections {
		if !strings.Contains(draft, s) {
			t.Errorf("draft missing section: %s", s)
		}
	}
}
