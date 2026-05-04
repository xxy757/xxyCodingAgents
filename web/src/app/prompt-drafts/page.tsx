// prompt-drafts/page.tsx - 提示词草稿
// 生成、编辑、发送结构化提示词草稿。
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
} from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";
import {
  Sparkle,
  PencilSimple,
  PaperPlaneRight,
  FloppyDisk,
  X,
  Lightning,
} from "@phosphor-icons/react/dist/ssr";

const TASK_TYPES = [
  { value: "", label: "自动推断" },
  { value: "bugfix", label: "修复 Bug" },
  { value: "build", label: "创建功能" },
  { value: "review", label: "代码审查" },
  { value: "qa", label: "测试验证" },
  { value: "docs", label: "写文档" },
  { value: "architecture", label: "架构设计" },
];

export default function PromptDraftsPage() {
  return (
    <Suspense fallback={<div className="text-sm text-zinc-400 py-12 text-center">加载中...</div>}>
      <PromptDraftsContent />
    </Suspense>
  );
}

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

  useEffect(() => {
    apiFetch<Project[]>("/api/projects")
      .then((p) => {
        setProjects(p);
        if (p.length > 0) setSelectedProjectId(p[0].id);
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (!selectedProjectId) return;
    listPromptDrafts(selectedProjectId).then(setDrafts).catch(() => setDrafts([]));
  }, [selectedProjectId]);

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

  const handleSend = async (draftId: string) => {
    if (editingDraft?.id === draftId && editContent !== (editingDraft.final_prompt || editingDraft.generated_prompt)) {
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

  return (
    <div className="space-y-6 animate-fade-up max-w-4xl">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">提示词草稿</h1>
        <p className="text-sm text-zinc-500 mt-1">生成、编辑并发送结构化提示词</p>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200/60 rounded-xl text-red-700 text-sm">{error}</div>
      )}
      {success && (
        <div className="p-3 bg-accent-50 border border-accent-200/60 rounded-xl text-accent-700 text-sm">{success}</div>
      )}

      {/* 输入区 */}
      <div className="card-bezel p-6">
        <div className="flex items-center gap-2 mb-4">
          <Lightning weight="fill" className="w-4 h-4 text-accent-600" />
          <span className="text-sm font-medium text-zinc-700">生成草稿</span>
        </div>
        <div className="flex gap-3 mb-3">
          <select
            value={selectedProjectId}
            onChange={(e) => setSelectedProjectId(e.target.value)}
            className="bg-zinc-50 border border-zinc-200 rounded-xl px-3 py-2.5 text-sm
                       focus:outline-none focus:ring-2 focus:ring-accent-500/20 focus:border-accent-500 transition-all"
          >
            {projects.map((p) => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>
          <select
            value={taskType}
            onChange={(e) => setTaskType(e.target.value)}
            className="bg-zinc-50 border border-zinc-200 rounded-xl px-3 py-2.5 text-sm
                       focus:outline-none focus:ring-2 focus:ring-accent-500/20 focus:border-accent-500 transition-all"
          >
            {TASK_TYPES.map((t) => (
              <option key={t.value} value={t.value}>{t.label}</option>
            ))}
          </select>
        </div>
        <textarea
          value={originalInput}
          onChange={(e) => setOriginalInput(e.target.value)}
          placeholder="描述你想要完成的任务..."
          className="w-full bg-zinc-50 border border-zinc-200 rounded-xl p-4 text-sm resize-y min-h-[80px]
                     placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-accent-500/20
                     focus:border-accent-500 transition-all"
          rows={3}
        />
        <div className="mt-3 flex gap-2 flex-wrap items-center">
          <button
            onClick={handleGenerate}
            disabled={loading || !originalInput.trim()}
            className="flex items-center gap-2 px-4 py-2.5 bg-zinc-900 text-white rounded-xl text-sm font-medium
                       hover:bg-zinc-800 disabled:opacity-50 pressable transition-colors"
          >
            <Sparkle className="w-4 h-4" />
            {loading ? "生成中..." : "生成草稿"}
          </button>
          {["bugfix", "build", "review", "qa", "docs"].map((t) => (
            <button
              key={t}
              onClick={() => setTaskType(t)}
              className={`px-3 py-2 text-xs font-medium rounded-lg border transition-all duration-200 pressable ${
                taskType === t
                  ? "bg-accent-50 border-accent-200 text-accent-700"
                  : "bg-zinc-50 border-zinc-200 text-zinc-500 hover:bg-zinc-100"
              }`}
            >
              {TASK_TYPES.find((tt) => tt.value === t)?.label}
            </button>
          ))}
        </div>
      </div>

      {/* 编辑区 */}
      {editingDraft && (
        <div className="card-bezel p-6 border-l-4 border-l-accent-500">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <PencilSimple className="w-4 h-4 text-zinc-500" />
              <h2 className="text-sm font-semibold text-zinc-900">编辑草稿</h2>
            </div>
            <StatusBadge status={editingDraft.status} />
          </div>
          <div className="text-xs text-zinc-400 mb-3">
            {editingDraft.task_type} | {editingDraft.original_input.slice(0, 60)}...
          </div>
          <textarea
            value={editContent}
            onChange={(e) => setEditContent(e.target.value)}
            className="w-full bg-zinc-50 border border-zinc-200 rounded-xl p-4 text-sm font-mono resize-y min-h-[200px]
                       focus:outline-none focus:ring-2 focus:ring-accent-500/20 focus:border-accent-500 transition-all"
            rows={10}
          />
          <div className="mt-4 flex gap-2">
            <button
              onClick={handleSave}
              disabled={loading}
              className="flex items-center gap-2 px-4 py-2.5 bg-zinc-700 text-white rounded-xl text-sm font-medium
                         hover:bg-zinc-600 disabled:opacity-50 pressable transition-colors"
            >
              <FloppyDisk className="w-4 h-4" />
              保存
            </button>
            <button
              onClick={() => handleSend(editingDraft.id)}
              disabled={loading}
              className="flex items-center gap-2 px-4 py-2.5 bg-accent-600 text-white rounded-xl text-sm font-medium
                         hover:bg-accent-700 disabled:opacity-50 pressable transition-colors"
            >
              <PaperPlaneRight className="w-4 h-4" />
              确认发送
            </button>
            <button
              onClick={() => setEditingDraft(null)}
              className="flex items-center gap-2 px-4 py-2.5 border border-zinc-200 rounded-xl text-sm text-zinc-600
                         hover:bg-zinc-50 pressable transition-colors"
            >
              <X className="w-4 h-4" />
              取消
            </button>
          </div>
        </div>
      )}

      {/* 草稿历史 */}
      <div className="card-bezel overflow-hidden">
        <div className="px-6 py-4 border-b border-zinc-100">
          <h2 className="text-sm font-semibold text-zinc-900">草稿历史</h2>
        </div>
        {drafts.length === 0 ? (
          <div className="p-12 text-center">
            <p className="text-sm text-zinc-400">暂无草稿</p>
          </div>
        ) : (
          <div className="divide-y divide-zinc-100">
            {drafts.map((draft) => (
              <div
                key={draft.id}
                className={`px-6 py-4 flex items-center justify-between transition-colors duration-200 ${
                  draft.status === "draft" ? "hover:bg-zinc-50 cursor-pointer" : "opacity-60"
                }`}
                onClick={() => {
                  if (draft.status === "draft") {
                    setEditingDraft(draft);
                    setEditContent(draft.final_prompt || draft.generated_prompt);
                  }
                }}
              >
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-zinc-900 truncate">{draft.original_input}</p>
                  <p className="text-xs text-zinc-400 mt-1">
                    {draft.task_type} | {new Date(draft.created_at).toLocaleString("zh-CN")}
                  </p>
                </div>
                <div className="flex items-center gap-3 shrink-0 ml-4">
                  <StatusBadge status={draft.status} />
                  {draft.status === "draft" && (
                    <button
                      onClick={(e) => { e.stopPropagation(); handleSend(draft.id); }}
                      className="flex items-center gap-1.5 px-3 py-2 text-xs font-medium bg-accent-600 text-white
                                 rounded-lg hover:bg-accent-700 pressable transition-colors"
                    >
                      <PaperPlaneRight className="w-3 h-3" />
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
