// prompt-drafts/page.tsx - 提示词草稿管理页面
// 支持生成结构化草稿、编辑确认、发送创建 Run/Task。
"use client";

import { useState, useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
  apiFetch,
  type Project,
  type PromptDraft,
  generatePromptDraft,
  updatePromptDraft,
  sendPromptDraft,
  listPromptDrafts,
  draftStatusColors,
} from "@/lib/api";

// 任务类型选项
const TASK_TYPES = [
  { value: "", label: "自动推断" },
  { value: "bugfix", label: "修复Bug" },
  { value: "build", label: "创建功能" },
  { value: "review", label: "代码审查" },
  { value: "qa", label: "测试验证" },
  { value: "docs", label: "写文档" },
  { value: "architecture", label: "架构设计" },
];

// PromptDraftsPageWrapper 包装 Suspense 边界以支持 useSearchParams
export default function PromptDraftsPage() {
  return (
    <Suspense fallback={<div className="p-6 text-center text-neutral-400">加载中...</div>}>
      <PromptDraftsContent />
    </Suspense>
  );
}

// PromptDraftsContent 提示词草稿管理页面主体
function PromptDraftsContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState("");
  const [drafts, setDrafts] = useState<PromptDraft[]>([]);
  const [originalInput, setOriginalInput] = useState(searchParams.get("input") || "");
  const [taskType, setTaskType] = useState(searchParams.get("type") || "");
  const [editingDraft, setEditingDraft] = useState<PromptDraft | null>(null);
  const [editContent, setEditContent] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // 加载项目列表
  useEffect(() => {
    apiFetch<Project[]>("/api/projects")
      .then((p) => {
        setProjects(p);
        if (p.length > 0) setSelectedProjectId(p[0].id);
      })
      .catch(() => {});
  }, []);

  // 加载草稿列表
  useEffect(() => {
    if (!selectedProjectId) return;
    listPromptDrafts(selectedProjectId)
      .then(setDrafts)
      .catch(() => setDrafts([]));
  }, [selectedProjectId]);

  // handleGenerate 生成草稿
  const handleGenerate = async () => {
    if (!selectedProjectId || !originalInput.trim()) return;
    setLoading(true);
    setError(null);
    setSuccess(null);
    try {
      const draft = await generatePromptDraft(selectedProjectId, originalInput, taskType || undefined);
      setDrafts([draft, ...drafts]);
      setEditingDraft(draft);
      setEditContent(draft.generated_prompt);
      setOriginalInput("");
      setSuccess("草稿已生成，请编辑后确认发送");
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  // handleSave 保存编辑
  const handleSave = async () => {
    if (!editingDraft) return;
    setLoading(true);
    setError(null);
    try {
      const updated = await updatePromptDraft(editingDraft.id, editContent, editingDraft.task_type);
      setDrafts(drafts.map((d) => (d.id === updated.id ? updated : d)));
      setEditingDraft(updated);
      setSuccess("草稿已保存");
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  // handleSend 发送草稿（发送前自动保存未提交的编辑）
  const handleSend = async (draftId: string) => {
    // 如果正在编辑的草稿就是要发送的草稿，先保存
    if (editingDraft && editingDraft.id === draftId && editContent !== (editingDraft.final_prompt || editingDraft.generated_prompt)) {
      try {
        await updatePromptDraft(editingDraft.id, editContent, editingDraft.task_type);
      } catch {
        setError("自动保存失败，请手动保存后再发送");
        return;
      }
    }
    setLoading(true);
    setError(null);
    setSuccess(null);
    try {
      const result = await sendPromptDraft(draftId);
      setDrafts(drafts.map((d) => (d.id === draftId ? { ...d, status: "sent", run_id: result.run_id } : d)));
      setEditingDraft(null);
      setSuccess(`已发送！Run: ${result.run_id}`);
      setTimeout(() => router.push("/runs"), 2000);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  // handleSelectDraft 选择草稿进行编辑
  const handleSelectDraft = (draft: PromptDraft) => {
    if (draft.status !== "draft") return;
    setEditingDraft(draft);
    setEditContent(draft.final_prompt || draft.generated_prompt);
  };

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">提示词草稿</h1>

      {/* 错误/成功提示 */}
      {error && (
        <div role="alert" className="mb-4 p-3 bg-error-50 text-error-700 rounded-md border border-error-500">
          操作失败：{error}
        </div>
      )}
      {success && (
        <div role="status" className="mb-4 p-3 bg-success-50 text-success-700 rounded-md border border-success-500">
          {success}
        </div>
      )}

      {/* 输入区 */}
      <div className="bg-white rounded-lg border border-neutral-200 p-4 mb-6">
        <div className="flex gap-4 mb-3">
          <select
            value={selectedProjectId}
            onChange={(e) => setSelectedProjectId(e.target.value)}
            aria-label="选择项目"
            className="border border-neutral-300 rounded-md px-3 py-2 text-sm flex-shrink-0 focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
          >
            {projects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
          <select
            value={taskType}
            onChange={(e) => setTaskType(e.target.value)}
            aria-label="任务类型"
            className="border border-neutral-300 rounded-md px-3 py-2 text-sm flex-shrink-0 focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
          >
            {TASK_TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </div>
        <textarea
          value={originalInput}
          onChange={(e) => setOriginalInput(e.target.value)}
          placeholder="描述你想要完成的任务..."
          aria-label="任务描述"
          className="w-full border border-neutral-300 rounded-md p-3 text-sm resize-y min-h-[80px] focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
          rows={3}
        />
        <div className="mt-3 flex gap-2 flex-wrap">
          <button
            onClick={handleGenerate}
            disabled={loading || !originalInput.trim()}
            className="px-4 py-2 bg-primary-600 text-white rounded-md hover:bg-primary-700 disabled:opacity-50 text-sm focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
          >
            {loading ? "生成中..." : "生成草稿"}
          </button>
          {/* 快捷任务类型按钮 */}
          {["bugfix", "build", "review", "qa", "docs"].map((t) => (
            <button
              key={t}
              onClick={() => { setTaskType(t); }}
              aria-label={TASK_TYPES.find((tt) => tt.value === t)?.label}
              className={`px-3 py-2 text-xs rounded-md border transition-colors focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none ${
                taskType === t
                  ? "bg-primary-50 border-primary-300 text-primary-700"
                  : "bg-neutral-50 border-neutral-200 hover:bg-neutral-100"
              }`}
            >
              {TASK_TYPES.find((tt) => tt.value === t)?.label}
            </button>
          ))}
        </div>
      </div>

      {/* 编辑区 */}
      {editingDraft && (
        <div className="bg-white rounded-lg border border-neutral-200 p-4 mb-6 border-l-4 border-l-primary-500">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-semibold">编辑草稿</h2>
            <span className={`px-2 py-1 text-xs rounded-full ${draftStatusColors[editingDraft.status] || "bg-neutral-100"}`}>
              {editingDraft.status === "draft" ? "草稿" : "已发送"}
            </span>
          </div>
          <div className="text-xs text-neutral-500 mb-2">
            任务类型: {editingDraft.task_type} | 原始输入: {editingDraft.original_input.slice(0, 60)}...
          </div>
          <textarea
            value={editContent}
            onChange={(e) => setEditContent(e.target.value)}
            aria-label="编辑草稿内容"
            className="w-full border border-neutral-300 rounded-md p-3 text-sm font-mono resize-y min-h-[200px] focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
            rows={10}
          />
          <div className="mt-3 flex gap-2">
            <button
              onClick={handleSave}
              disabled={loading}
              className="px-4 py-2 bg-neutral-600 text-white rounded-md hover:bg-neutral-700 disabled:opacity-50 text-sm focus-visible:ring-2 focus-visible:ring-neutral-400 focus-visible:outline-none"
            >
              保存
            </button>
            <button
              onClick={() => handleSend(editingDraft.id)}
              disabled={loading}
              className="px-4 py-2 bg-success-600 text-white rounded-md hover:bg-success-700 disabled:opacity-50 text-sm focus-visible:ring-2 focus-visible:ring-success-500 focus-visible:outline-none"
            >
              确认发送
            </button>
            <button
              onClick={() => setEditingDraft(null)}
              className="px-4 py-2 border border-neutral-300 rounded-md hover:bg-neutral-50 text-sm focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
            >
              取消
            </button>
          </div>
        </div>
      )}

      {/* 草稿历史列表 */}
      <div className="bg-white rounded-lg border border-neutral-200">
        <div className="px-4 py-3 border-b border-neutral-200">
          <h2 className="font-semibold">草稿历史</h2>
        </div>
        {drafts.length === 0 ? (
          <div className="p-8 text-center text-neutral-400">暂无草稿</div>
        ) : (
          <div className="divide-y divide-neutral-100">
            {drafts.map((draft) => (
              <div
                key={draft.id}
                className={`px-4 py-3 flex items-center justify-between transition-colors ${
                  draft.status === "draft" ? "hover:bg-neutral-50 cursor-pointer" : "opacity-60"
                }`}
                onClick={() => handleSelectDraft(draft)}
                role={draft.status === "draft" ? "button" : undefined}
                tabIndex={draft.status === "draft" ? 0 : undefined}
                onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") handleSelectDraft(draft); }}
              >
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate">{draft.original_input}</div>
                  <div className="text-xs text-neutral-500 mt-1">
                    {draft.task_type} | {new Date(draft.created_at).toLocaleString("zh-CN")}
                  </div>
                </div>
                <div className="flex items-center gap-2 ml-4">
                  <span className={`px-2 py-1.5 text-xs rounded-full ${draftStatusColors[draft.status] || "bg-neutral-100"}`}>
                    {draft.status === "draft" ? "草稿" : "已发送"}
                  </span>
                  {draft.status === "draft" && (
                    <button
                      onClick={(e) => { e.stopPropagation(); handleSend(draft.id); }}
                      aria-label={`发送草稿: ${draft.original_input.slice(0, 20)}`}
                      className="px-3 py-2 text-sm bg-success-600 text-white rounded-md hover:bg-success-700 focus-visible:ring-2 focus-visible:ring-success-500 focus-visible:outline-none"
                    >
                      发送
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
